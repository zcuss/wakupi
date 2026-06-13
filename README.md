# Wakupi

<h4 align="center">
  WhatsApp Desktop Client with AI, Dynamic QRIS, and Desktop Control
</h4>

<p align="center">
  <img src="https://img.shields.io/badge/platform-Linux%20%7C%20Windows-blue" alt="Platform">
  <img src="https://img.shields.io/badge/go-1.25%2B-00ADD8?logo=go" alt="Go">
  <img src="https://img.shields.io/badge/vue-3.x-4FC08D?logo=vue.js" alt="Vue">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
</p>

---

## ✨ Features

### 💬 WhatsApp Multi-Account
- Multi-session WhatsApp Web (multi-device)
- Real-time messaging with end-to-end encryption
- Media support: images, video, audio, documents, stickers
- Voice notes recording & playback
- Status/Story viewer & poster
- Group management (create, add/remove, promote/demote)
- Chat actions: pin, archive, mute, block, star, search, forward

### 🤖 AI Assistant
- Multi-provider: OpenAI, Anthropic (Claude), Google (Gemini), Ollama (local)
- AI Playground with streaming responses
- Smart reply suggestions
- Message summarization
- Tone-based message composition
- Per-chat AI toggle & settings

### 💳 Dynamic QRIS Generator
- Upload your QRIS static QR code
- Generate dynamic QRIS with custom amounts
- Send QRIS invoices directly to WhatsApp chats
- Transaction history with status tracking
- Product catalog for quick invoicing
- Instant invoice generator in chat

### 🖥️ Desktop Controller
- View running apps
- Quick launch applications (Terminal, Browser, Files, etc.)
- Media controls (Play/Pause, Next, Previous)
- Volume control slider
- Screenshot capture
- Screen lock
- **WhatsApp remote control** — send `!open terminal`, `!play`, `!volume 80`, etc. from another phone

### 🎨 Modern UI
- WhatsApp-inspired design
- Light/Dark/System theme
- Tailwind CSS
- Emoji picker
- Context menus

---

## 🚀 Quick Start

### Linux

```bash
# Clone
git clone https://github.com/hirotomasato/wakupi.git
cd wakupi

# Build
wails build -tags webkit2_41

# Run
./build/bin/wakupi
```

**Requirements:**
- Go 1.25+
- Node.js 18+
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- WebKit2GTK 4.1 (`sudo apt install libwebkit2gtk-4.1-dev`)

### Windows

```bash
# Clone
git clone https://github.com/hirotomasato/wakupi.git
cd wakupi

# Build
wails build

# Run
./build/bin/wakupi.exe
```

---

## ⌨️ Desktop Commands (WhatsApp Remote)

Send these commands from any WhatsApp chat:

| Command | Action |
|---------|--------|
| `!open terminal` | Launch application |
| `!close firefox` | Close application |
| `!apps` | List running apps |
| `!play` / `!pause` | Play/Pause media |
| `!next` / `!prev` | Skip track |
| `!now` | Show current track |
| `!volume 80` | Set volume (0-100) |
| `!screenshot` | Capture screenshot |
| `!lock` | Lock screen |
| `!help` | Show all commands |

---

## 🌍 Platform Support

| Feature | Linux | Windows |
|---------|-------|---------|
| WhatsApp Multi-Account | ✅ | ✅ |
| AI Assistant | ✅ | ✅ |
| QRIS Generator | ✅ | ✅ |
| Desktop Controller | ✅ D-Bus | ✅ PowerShell |
| Media Controls | ✅ MPRIS2 | ✅ SMTC |

> macOS support planned for future releases.

---

## 🛠️ Tech Stack

| Layer | Technology |
|-------|-----------|
| **Desktop Shell** | [Wails v2](https://wails.io) |
| **Backend** | Go 1.25 |
| **Frontend** | Vue 3 + TypeScript + Pinia |
| **Styling** | Tailwind CSS |
| **WhatsApp** | [whatsmeow](https://github.com/tulir/whatsmeow) |
| **AI** | OpenAI / Anthropic / Gemini / Ollama |
| **Database** | SQLite |
| **D-Bus** | godbus/v5 |
| **QR Code** | qrcode + jsQR |

---

## 📁 Project Structure

```
wakupi/
├── app.go              # Wails bindings (Go→JS bridge)
├── main.go             # App entry point
├── internal/
│   ├── ai/             # AI provider abstraction
│   ├── desktop/        # Desktop controller (Linux+Windows)
│   └── wa/             # WhatsApp manager
│       ├── manager.go  # Session & event routing
│       ├── messages.go # Inbound message handling
│       ├── send.go     # Outbound message sending
│       ├── actions.go  # Chat actions & groups
│       ├── avatar.go   # Profile picture caching
│       └── store.go    # SQLite persistence
├── frontend/
│   └── src/
│       ├── components/ # Vue components
│       ├── stores/     # Pinia stores
│       ├── lib/        # Shared utilities (QRIS, etc.)
│       └── style.css   # Tailwind styles
└── data/               # Local WhatsApp data & media
```

---

## 🤝 Contributing

Contributions are welcome! Feel free to open issues and pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

---

## 📄 License

MIT © [Masanto](https://github.com/hirotomasato)

See [LICENSE](LICENSE) for full text.

---

## ⚠️ Disclaimer

This project is not affiliated with WhatsApp (Meta). Use at your own risk. WhatsApp may ban accounts that use unofficial clients — always use a secondary number for testing.
