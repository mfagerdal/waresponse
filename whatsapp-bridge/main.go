package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
)

var (
	openaiClient *openai.Client
	client       *whatsmeow.Client

	// For tracking pending responses
	pendingResponses = make(map[string]*time.Timer)
	pendingMutex     = sync.RWMutex{}
	userJID          types.JID // Will be set when we connect
)

func main() {
	// Load environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Initialize OpenAI client
	openaiClient = openai.NewClient(apiKey)

	// Initialize WhatsApp client
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:store/whatsapp.db?_foreign_keys=on", dbLog)
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get device: %v", err)
	}

	clientLog := waLog.Stdout("Client", "INFO", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)

	// Add event handlers
	client.AddEventHandler(handleMessage)
	client.AddEventHandler(handleConnection)

	// Connect to WhatsApp
	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("📱 Scan this QR code with WhatsApp:")
				qrcode := evt.Code
				printQR(qrcode)
			} else {
				fmt.Printf("Login event: %s\n", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
	}

	fmt.Println("✅ WhatsApp connected successfully!")
	fmt.Println("🤖 Bot is now active and monitoring family groups...")

	// Wait for interrupt signal to gracefully shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n👋 Shutting down gracefully...")
	client.Disconnect()
}

func handleMessage(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// Skip if message is from us
		if v.Info.IsFromMe {
			log.Printf("📝 Skipping message from self")
			return
		}

		// Debug: Log all message types to understand what we're receiving
		log.Printf("📝 Received message - IsGroup: %v, MessageType: %T", v.Info.IsGroup, v.Message)

		// Only handle group messages
		if !v.Info.IsGroup {
			log.Printf("📝 Received private message, ignoring")
			return
		}

		// Get message text with better extraction
		text := ""
		if v.Message.GetConversation() != "" {
			text = v.Message.GetConversation()
		} else if v.Message.GetExtendedTextMessage() != nil {
			text = v.Message.GetExtendedTextMessage().GetText()
		} else if v.Message.GetImageMessage() != nil && v.Message.GetImageMessage().GetCaption() != "" {
			text = v.Message.GetImageMessage().GetCaption()
		} else if v.Message.GetVideoMessage() != nil && v.Message.GetVideoMessage().GetCaption() != "" {
			text = v.Message.GetVideoMessage().GetCaption()
		}

		// Log what we extracted
		log.Printf("📝 Extracted text: '%s' (length: %d)", text, len(text))

		if text == "" {
			log.Printf("📝 No text content found in message, ignoring")
			return
		}

		// Check if group name contains "family"
		groupInfo, err := client.GetGroupInfo(v.Info.Chat)
		if err != nil {
			log.Printf("❌ Failed to get group info: %v", err)
			return
		}

		groupName := strings.ToLower(groupInfo.Name)
		log.Printf("📝 Message from group: '%s', text: '%s'", groupName, text)

		if !strings.Contains(groupName, "family") {
			log.Printf("📝 Group '%s' doesn't contain 'family', ignoring", groupName)
			return
		}

		// Check if message should trigger food auto-response
		shouldRespond := shouldAutoRespond(text)
		log.Printf("📝 Should respond to '%s': %v", text, shouldRespond)

		if shouldRespond {
			log.Printf("🤖 Triggering auto-reply for: '%s'", text)
			// Generate and send reply
			go sendAutoReply(v.Info.Chat, text)
			return // Don't set up waiting response for food triggers
		}

		// Check if this is a message from someone else (not us) in a family group
		if !v.Info.IsFromMe {
			// Cancel any existing timer for this chat
			cancelPendingResponse(v.Info.Chat.String())

			// Set up auto-response timer (10 seconds)
			log.Printf("⏰ Setting 10s timer for auto-response in group: %s", groupName)
			setPendingResponse(v.Info.Chat, 10*time.Second)
		} else {
			// If this is from us, cancel any pending response timers for this chat
			log.Printf("📱 User responded, canceling auto-response timer")
			cancelPendingResponse(v.Info.Chat.String())
		}
	}
}

func shouldAutoRespond(text string) bool {
	// Match dinner-like questions (case insensitive) in multiple languages
	pattern := `(?i)(sushi|pizza|dinner|takeout|eat|hungry|food|restaurant|order|mat|middag|lunch|äta|hungrig|käk|maten|restaurang|beställa|när.*mat|vad.*äta)`
	matched, _ := regexp.MatchString(pattern, text)
	log.Printf("🔍 Checking text '%s' against pattern - matched: %v", text, matched)
	return matched
}

func sendAutoReply(chatJID types.JID, originalMessage string) {
	// Generate ChatGPT response
	resp, err := openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "system",
					Content: "You are a cheerful auto-responder in Marcus's family WhatsApp chat. Say yes to food questions unless otherwise instructed. Keep responses short and family-friendly.",
				},
				{
					Role:    "user",
					Content: originalMessage,
				},
			},
			MaxTokens: 150,
		},
	)

	if err != nil {
		log.Printf("ChatGPT error: %v", err)
		return
	}

	if len(resp.Choices) == 0 {
		log.Printf("No response from ChatGPT")
		return
	}

	reply := resp.Choices[0].Message.Content

	// Send the reply
	_, err = client.SendMessage(context.Background(), chatJID, &waE2E.Message{
		Conversation: proto.String(reply),
	})

	if err != nil {
		log.Printf("Failed to send message: %v", err)
	} else {
		log.Printf("🤖 Auto-replied: %s", reply)
	}
}

func handleConnection(evt interface{}) {
	switch evt.(type) {
	case *events.Connected:
		log.Printf("✅ Connected to WhatsApp!")
		// Store our JID for later use
		if client != nil && client.Store.ID != nil {
			userJID = *client.Store.ID
			log.Printf("📱 Our JID: %s", userJID)
		}
	case *events.StreamReplaced:
		log.Printf("🔄 Stream replaced, reconnecting...")
	case *events.Disconnected:
		log.Printf("❌ Disconnected from WhatsApp")
	case *events.LoggedOut:
		log.Printf("🚪 Logged out from WhatsApp")
	}
}

func setPendingResponse(chatJID types.JID, delay time.Duration) {
	pendingMutex.Lock()
	defer pendingMutex.Unlock()

	chatKey := chatJID.String()

	// Cancel existing timer if any
	if timer, exists := pendingResponses[chatKey]; exists {
		timer.Stop()
	}

	// Set new timer
	timer := time.AfterFunc(delay, func() {
		log.Printf("⏰ Timer expired! Sending waiting message to %s", chatKey)
		sendWaitingMessage(chatJID)

		// Clean up
		pendingMutex.Lock()
		delete(pendingResponses, chatKey)
		pendingMutex.Unlock()
	})

	pendingResponses[chatKey] = timer
}

func cancelPendingResponse(chatKey string) {
	pendingMutex.Lock()
	defer pendingMutex.Unlock()

	if timer, exists := pendingResponses[chatKey]; exists {
		timer.Stop()
		delete(pendingResponses, chatKey)
		log.Printf("✅ Canceled auto-response timer for %s", chatKey)
	}
}

func sendWaitingMessage(chatJID types.JID) {
	waitingMessage := "Jag svarar snart." // "I will answer soon" in Swedish

	_, err := client.SendMessage(context.Background(), chatJID, &waE2E.Message{
		Conversation: proto.String(waitingMessage),
	})

	if err != nil {
		log.Printf("❌ Failed to send waiting message: %v", err)
	} else {
		log.Printf("💬 Sent waiting message: %s", waitingMessage)
	}
}

// Simple QR code printer (basic ASCII representation)
func printQR(code string) {
	// This is a very basic implementation
	// For a proper QR code, you might want to use a dedicated library
	fmt.Printf("QR Code Data: %s\n", code)
	fmt.Println("Please scan this with your WhatsApp mobile app")
}
