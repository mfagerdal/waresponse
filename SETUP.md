# WhatsApp Family Responder - Setup Guide

## 🚀 Quick Setup

### 1. Environment Variables
Create a `.env` file in the root directory:
```env
OPENAI_API_KEY=sk-your-actual-openai-key-here
```

### 2. Install Dependencies
Make sure Go is installed (>=1.21):
```bash
# On macOS with Homebrew
brew install go

# Navigate to the whatsapp-bridge directory
cd whatsapp-bridge
go mod tidy
```

### 3. Run Locally
```bash
cd whatsapp-bridge
go run main.go
```
- Scan the QR code with your WhatsApp
- Bot will automatically respond in family groups to food-related messages

## 🌐 Railway Deployment

### 1. Prepare Repository
```bash
git init
git add .
git commit -m "Initial commit"
git remote add origin your-github-repo-url
git push -u origin main
```

### 2. Deploy to Railway
1. Go to [Railway.app](https://railway.app)
2. Connect your GitHub account
3. Import your repository
4. Add environment variable:
   - Name: `OPENAI_API_KEY`
   - Value: Your OpenAI API key
5. Deploy!

### 3. Initial Authentication
After deployment:
1. Check Railway logs for the QR code
2. Scan it with WhatsApp on your phone
3. The bot will stay authenticated

## 🔧 Configuration

### Group Detection
The bot only responds in groups with "family" in the name (case-insensitive).

### Trigger Words
Bot responds to messages containing:
- sushi
- pizza  
- dinner
- takeout
- eat
- hungry

### Personality
The bot acts as a cheerful family member who generally says yes to food questions.

## 📝 Customization

### Change Trigger Words
Edit `src/events/message.js`, line 25:
```js
const shouldAutoRespond = /sushi|pizza|dinner|takeout|eat|hungry/i.test(text);
```

### Change Personality
Edit `utils/chatgpt.js`, line 11:
```js
{ role: "system", content: "Your custom personality here" }
```

### Change Group Filter
Edit `src/events/message.js`, line 22:
```js
if (!groupName.includes("family")) return;
```

## 🛡️ Security Notes

- Never commit your `.env` file
- Use Railway's environment variables for production
- The bot only responds in groups, not private messages
- WhatsApp session is stored locally in `auth_info_baileys/`

## 🐛 Troubleshooting

### QR Code Not Showing
- Check that `OPENAI_API_KEY` is set
- Ensure Node.js version >= 18
- Try restarting the bot

### Bot Not Responding
- Check that group name contains "family"
- Verify message contains trigger words
- Check Railway logs for errors

### Connection Issues
- Bot auto-reconnects on connection loss
- May need to re-scan QR code after extended downtime
- Check Railway logs for connection status