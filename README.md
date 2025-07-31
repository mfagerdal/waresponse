# 🤖 WhatsApp Family Responder

This project automatically replies in your WhatsApp **family group** using ChatGPT (GPT-4) via the [whatsmeow](https://github.com/tulir/whatsmeow) Go library, based on the [whatsapp-mcp](https://github.com/lharries/whatsapp-mcp) architecture.

---

## 🚀 Features

- Auto-replies to group messages like "Do you want sushi?"
- Uses OpenAI's GPT-4 API for smart and fun answers
- Only replies in groups with names containing `family`
- Built with Go for reliability and 24/7 deployment
- Easy to deploy locally or to Railway/Render

---

## 📦 Prerequisites

- Go (>=1.21) - Install with `brew install go` on macOS
- OpenAI API key: [https://platform.openai.com/account/api-keys](https://platform.openai.com/account/api-keys)
- WhatsApp account for scanning the QR code

---

## 🛠 Setup

### 1. Environment Variables

Create a `.env` file in the root directory:

```env
OPENAI_API_KEY=sk-your-openai-key-here
```

### 2. Install Dependencies

```bash
cd whatsapp-bridge
go mod tidy
```

---

## ▶️ Run It

**Easy way:**
```bash
./start.sh
```

**Manual way:**
```bash
cd whatsapp-bridge
go run main.go
```

Scan the QR code from WhatsApp on your phone. You're now live!

---

## 🌐 Deploy to Cloud (Optional)

Use [Railway](https://railway.app) or [Render](https://render.com) to keep the bot running 24/7:

1. Push your repo to GitHub
2. Import it into Railway/Render
3. Add your `OPENAI_API_KEY` as an environment variable
4. Deploy!

---

## ✅ Summary

- Replies in family group chats
- Triggers on casual dinner/food messages
- GPT-4 powered personality replies
- Deployable in cloud or run locally

Happy hacking! 🍣
