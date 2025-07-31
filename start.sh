#!/bin/bash

# WhatsApp Family Responder Startup Script

echo "🚀 Starting WhatsApp Family Responder..."

# Check if .env file exists
if [ ! -f ".env" ]; then
    echo "❌ .env file not found!"
    echo "Please create a .env file with your OPENAI_API_KEY"
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed!"
    echo "Please install Go: brew install go"
    exit 1
fi

# Navigate to whatsapp-bridge directory
cd whatsapp-bridge || {
    echo "❌ whatsapp-bridge directory not found!"
    exit 1
}

# Build and run the application
echo "📦 Building application..."
go build -o whatsapp-bot main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "🤖 Starting WhatsApp Family Responder..."
    echo "📱 Get ready to scan the QR code with your WhatsApp..."
    ./whatsapp-bot
else
    echo "❌ Build failed!"
    exit 1
fi