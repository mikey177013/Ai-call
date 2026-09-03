# WhatsApp AI Call Assistant Bot (Golang)

An automated WhatsApp voice-call bot written in Go (Golang), powered by:
- **`whatsmeow` & `meowcaller`**: A pure-Go WhatsApp VOIP call protocol client (answering & placing calls, encoding/decoding MLow codec audio).
- **Groq API (`whisper-large-v3`)**: Ultra-fast speech transcription (Speech-to-Text).
- **Google Gemini API (`gemini-3.1-flash-lite`)**: The latest generation conversational AI — extremely fast (~0.3 second latency) and token-efficient for voice calls.
- **Multi-engine TTS (Gemini TTS / Edge TTS / ElevenLabs / OpenAI / Anime TTS)**: Realistic human speech synthesis with natural intonation, played directly into the WhatsApp phone channel.

---

## 📌 Features

1. ⚡ **Ultra-fast conversational AI (`gemini-3.1-flash-lite`)**: Uses Gemini 3.1 Flash Lite, which is very fast (~0.3s latency) and free of tight daily quotas, with an automatic *fallback* to other Gemini versions if a failure occurs.
2. 🔊 **Support for 5 different TTS engines**:
   - **Gemini Audio TTS (`TTS_ENGINE=geminitts`)**: Google Gemini Audio's native voices with natural breathing pauses (`Puck`, `Kore`, `Aoede`, etc.), plus a **NexRay Gemini TTS API fallback** if the official Gemini quota is exceeded.
   - **Microsoft Edge Neural TTS (`TTS_ENGINE=edgetts`)**: **100% FREE FOREVER** with no API key (`en-US-AvaMultilingualNeural`, `en-US-AndrewMultilingualNeural`, etc.).
   - **ElevenLabs Human Voice (`TTS_ENGINE=elevenlabs`)**: The most realistic human voices in the world with strong emotion (free 10,000 characters/month).
   - **OpenAI Human Voice (`TTS_ENGINE=openai`)**: Friendly-sounding OpenAI Audio voices (`nova`, `shimmer`, `alloy`, etc.).
   - **Anime VITS TTS (`TTS_ENGINE=animetts`)**: Japanese anime character voices (VITS).
3. 🎛️ **Voice pitch & speed customization**: Change the voice pitch (`TTS_PITCH=-2Hz`) and speaking speed (`TTS_SPEED=1.0`) via `.env` so the conversation feels relaxed and easy-going.
4. 🔐 **Owner security feature (`OWNER`)**: Restricts call and chat-command access to the phone numbers registered in `.env` (supports multiple owners separated by commas).
5. 💬 **Works in groups & DMs**: The `!aicall <number>` or `.call <number>` commands work seamlessly in direct messages and in groups (supports ephemeral, view-once, & extended text messages).
6. 🗄️ **SQLite database in WAL mode**: Uses SQLite with `WAL` mode and `busy_timeout` to prevent *database locking* when handling concurrent WhatsApp events.

---

## 🏗️ Voice Call Architecture

```
[Incoming Call / Call Command]
             │
             ▼
    [WhatsApp VOIP Connection] (whatsmeow / meowcaller)
             │
             ▼
    [Call Audio Recording]      (Buffering a 6-second audio stream)
             │
             ▼
    [Speech-to-Text (STT)]      (Groq Whisper Large v3)
             │
             ▼
    [Conversational AI Agent]   (Google Gemini 3.1 Flash Lite)
             │
             ▼
    [Speech Synthesis (TTS)]    (Gemini TTS / Edge TTS / ElevenLabs / OpenAI)
             │
             ▼
    [MLow Audio Encoding]       (Played into the WhatsApp phone channel)
```

---

## 🔗 How the Connection Actually Works (Code Analysis)

This is the part most people get stuck on, so here is exactly what the code does.

### 1. There is no "phone number" setting — the bot *becomes* your WhatsApp account

There is **no** `PHONE_NUMBER` variable anywhere in the config. The bot does not
"dial into" a number you configure. Instead, it registers itself as a **linked
device (WhatsApp Web companion)** on an existing WhatsApp account, exactly like
WhatsApp Web in a browser. Whatever number you scan the QR code with **becomes
the bot's number**.

The `OWNER` variable is *not* the bot's number — it is a **whitelist of numbers
allowed to command the bot** (see `IsOwner()` in `config.go`).

### 2. Session storage (`main.go`)

```go
container, err := sqlstore.New(context.Background(), "sqlite",
    "file:ai_call_session.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)...", dbLog)
deviceStore, err := container.GetFirstDevice(context.Background())
waClient := whatsmeow.NewClient(deviceStore, clientLog)
```

The login credentials live in the **`ai_call_session.db`** SQLite file next to the
binary. This single file *is* your login:

- If `waClient.Store.ID == nil` (no session yet) → the bot prints a **QR code** in
  the terminal for you to scan.
- If a session already exists → it reconnects silently and logs
  `Successfully connected to WhatsApp!`. **No QR code is shown again.**

> 🔴 **This is the single most important fact for hosting**: if the host wipes the
> filesystem on restart (Render/Railway without a persistent volume), `ai_call_session.db`
> is deleted and the bot demands a new QR scan on every deploy. Fix this with a
> persistent disk/volume — see the hosting section.

Note that `.gitignore` excludes `*.db`, so the session is **never** committed to git.

### 3. Linking the device — QR code (`main.go`)

```go
if waClient.Store.ID == nil {
    qrChan, _ := waClient.GetQRChannel(context.Background())
    err = waClient.Connect()
    for evt := range qrChan {
        if evt.Event == "code" {
            fmt.Println("\nScan the QR code below using WhatsApp on your phone:")
            qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
        }
    }
}
```

The QR is rendered as **half-block ASCII art directly into stdout**. So yes — you
must link a device, and you do it by scanning that terminal QR with the phone whose
number you want the bot to use. This implementation supports **QR linking only**
(there is no 8-digit pairing-code path in the code).

### 4. Placing an outgoing call (`main.go` → `caller.go`)

```go
targetJID := types.NewJID(targetNum, types.DefaultUserServer)  // 628xxx@s.whatsapp.net
call, err := callerClient.Call(ctx, targetJID.String())        // 45s timeout
call.OnReady(func() { go session.StartVoiceLoop() })           // fires when answered
```

When you send `.call 628123456789`, the bot normalizes the number (strips `+`,
spaces and `-`), converts it to a WhatsApp JID, and rings it over VOIP. The AI
voice loop only starts once `OnReady` fires — i.e. when the person actually picks up.

### 5. Receiving an incoming call (`main.go`)

```go
callerClient.OnIncomingCall(func(call *meowcaller.Call) {
    if !IsOwner(call.Peer().User) { _ = call.Reject(); return }
    call.Answer()
    go session.StartVoiceLoop()
})
```

Anyone who is **not** an owner gets their call **rejected automatically**. Owners
get answered and dropped straight into an AI conversation.

### 6. The conversation loop (`caller.go`)

Once connected, `StartVoiceLoop()` runs this cycle:

1. Plays a greeting: *"Hello! I am your AI assistant..."*
2. Records **6 seconds** of the caller's audio into `temp/rec_<callID>_<round>.wav`
   via `meowcaller.WAVRecorder`.
3. Discards the chunk if it is smaller than 4000 bytes (silence).
4. Transcribes it with Groq Whisper (`TranscribeAudio`, `stt.go`).
5. Sends the text to Gemini with the conversation history (`ai.go`).
6. Converts the reply to audio (`GenerateTTS`, `tts.go`) and plays it back through
   the MLow-encoded call channel.
7. Repeats until the call ends or `call.State() != CallPhaseActive`.

### Connection flow summary

```
Your phone (e.g. +62812xxxx)
   │  scans terminal QR via WhatsApp > Linked Devices
   ▼
ai_call_session.db  ← credentials persist here (MUST survive restarts)
   │
   ▼
Bot runs as a linked device of that number
   │
   ├── receives WhatsApp text commands  (!aicall / .call / !engine ...)
   └── places & answers VOIP calls      (meowcaller → MLow audio)
```

---

## 📋 Prerequisites

- **Go (Golang)**: Version 1.22 or newer (the module targets 1.25).
- **Node.js**: Version 18 or newer (required for the Microsoft Edge TTS engine).
- **API Keys**:
  - **Gemini API Key** (free): Get it at [Google AI Studio](https://aistudio.google.com/app/apikey)
  - **Groq API Key** (free): Get it at [Groq Console](https://console.groq.com/keys)
  - **ElevenLabs / OpenAI API Key** (optional)
- **A spare WhatsApp number** is strongly recommended. Automating calls can get a
  number banned — do not use your primary personal number.

---

## 🚀 Installation & Usage

### 1. Clone the repository
```bash
git clone https://github.com/krsna081/assisten-ai-call.git
cd assisten-ai-call
```

### 2. Install the Node.js dependency (for the Edge TTS engine)
```bash
npm install
```

### 3. Configure the environment variables (`.env`)
Copy the `.env.example` template to `.env`:
```bash
cp .env.example .env
```

Open the `.env` file and set your configuration:
```env
# Primary API keys (required)
GEMINI_API=AIzaSy...                # Your Gemini API key
GEMINI_MODEL=gemini-3.1-flash-lite  # Gemini model (default: gemini-3.1-flash-lite)
GROQ_API=gsk_...                    # Your Groq API key

# TTS engine (options: edgetts / geminitts / elevenlabs / openai / animetts / google)
TTS_ENGINE=edgetts

# Voice selection (e.g. en-US-AvaMultilingualNeural / en-US-AndrewMultilingualNeural / Puck)
TTS_VOICE=en-US-AvaMultilingualNeural

# Voice pitch & speed
TTS_PITCH=-1Hz
TTS_SPEED=1.0

# Owner whitelist (leave empty to make the bot public)
OWNER=628123456789,628987654321
```

### 4. Build and run the application
```bash
go build -o ai-call .
./ai-call
```

### 5. Link your WhatsApp number (scan the QR)

On the **first** run the terminal prints a QR code. On your phone:

1. Open **WhatsApp**
2. Tap **Settings** (or the ⋮ menu on Android)
3. Tap **Linked Devices**
4. Tap **Link a Device**
5. Point the camera at the QR code in your terminal

When you see `Successfully connected to WhatsApp!` the bot is live. The
`ai_call_session.db` file now holds the session — **back it up** and keep it, and
the bot will never ask for a QR again.

### 6. Make your first AI call

From the owner number, send a WhatsApp message (to the bot's own chat, any DM, or
a group the bot is in):

```
.call 628123456789
```

Use the **full international format with no `+` and no spaces** (e.g. `919812345678`
for India, `628123456789` for Indonesia). The bot replies
`Starting an AI call to ...`, rings the target, and once they answer the AI starts
talking.

---

## ☁️ Hosting Guide (Connect a Number 24/7)

The bot must stay online continuously to answer calls. In every case the flow is
the same: **deploy → open the console/logs → scan the QR once → keep
`ai_call_session.db` persistent.**

### ⚠️ Two universal rules for every host

1. **You need console/log access to scan the QR.** The QR is ASCII art printed to
   stdout. Hosts that hide stdout cannot be linked directly (see the workaround below).
2. **`ai_call_session.db` must survive restarts.** On an ephemeral filesystem you
   will be forced to re-scan the QR after every restart or redeploy.

### 💡 Workaround: link locally, then upload the session

This is the most reliable method and it works on **every** host, including those
with unreadable log rendering:

1. Run the bot **on your own PC** first (`go build -o ai-call . && ./ai-call`).
2. Scan the QR locally and wait for `Successfully connected to WhatsApp!`.
3. Stop the bot (`Ctrl+C`).
4. Upload the generated **`ai_call_session.db`** (plus `ai_call_session.db-wal`
   and `ai_call_session.db-shm` if present) to the host, in the same directory as
   the binary/project.
5. Start the bot on the host — it reconnects with the existing session and never
   shows a QR.

> Run the bot in **only one place at a time**. Two instances sharing one session
> will fight over the connection and may invalidate the device link.

---

### 1️⃣ Katabump (free panel hosting)

Katabump is a Pterodactyl-style game panel, which is ideal here because it gives
you a real interactive console and persistent files.

1. Create a server on Katabump. Choose the **Go** egg if it exists; otherwise pick
   a generic Ubuntu/Debian "Docker VPS"-style egg.
2. Open **File Manager** and upload your project (or use the console:
   `git clone https://github.com/krsna081/assisten-ai-call.git`).
3. In the console, install the toolchain if the egg does not provide it, then:
   ```bash
   npm install
   go build -o ai-call .
   ```
4. Create your `.env` in the File Manager (paste the contents from `.env.example`
   and fill in the keys). **Never upload a real `.env` to a public git repo.**
5. Set the startup command to `./ai-call`.
6. Press **Start**, open the **Console** tab, and **scan the QR code** shown there
   with *Linked Devices → Link a Device*.
7. Enable **auto-restart / "keep alive"** in the server settings so it survives crashes.

Katabump keeps files on disk between restarts, so `ai_call_session.db` persists
and you only scan once. If the console renders the QR poorly, use the
"link locally, then upload the session" workaround above.

---

### 2️⃣ BotHosting.net (free bot panel)

BotHosting also uses a Pterodactyl panel, so the steps mirror Katabump.

1. Create a server and select an egg with **Go** support (or a generic
   Ubuntu/"custom" egg where you can install Go).
2. Upload the project via **File Manager** → **Upload**, or clone it from the console.
3. Console:
   ```bash
   npm install
   go build -o ai-call .
   ```
4. Add the environment variables. You can either create the `.env` file in the File
   Manager, or use the panel's **Startup → Variables** tab if your egg exposes
   custom variables (the code reads real environment variables too, so both work).
5. Set the startup command to `./ai-call` and press **Start**.
6. Scan the QR from the **Console** tab.

> BotHosting free plans are usually limited in RAM/CPU. Give the bot at least
> **512 MB RAM**; MLow audio encoding plus the Node Edge TTS subprocess needs headroom.
> If Node cannot run on your plan, switch to `TTS_ENGINE=geminitts` (pure Go + HTTP,
> no Node required).

---

### 3️⃣ Render

Render is designed for web services, so there are two things to handle: the
**ephemeral disk** and the **absence of an interactive console** on free plans.

1. Push your project to GitHub (without `.env` and without the `.db` files).
2. On Render: **New → Background Worker** (not a Web Service — this bot exposes no
   HTTP port). Connect your repo.
3. Configure:
   - **Environment**: `Docker` (recommended, so you control Go + Node), or the
     native Go environment if you only use `geminitts`.
   - **Build Command**: `npm install && go build -o ai-call .`
   - **Start Command**: `./ai-call`
4. Add every variable from `.env.example` under **Environment → Environment Variables**
   (`GEMINI_API`, `GROQ_API`, `TTS_ENGINE`, `TTS_VOICE`, `OWNER`, ...).
5. 🔴 **Add a persistent disk**: **Settings → Disks → Add Disk**, mount path
   `/app` (or wherever the binary runs). Without a disk, Render wipes
   `ai_call_session.db` on every deploy and the bot will keep asking for a QR.
6. **Linking on Render**: free Render plans have **no shell**, so read the QR from
   the **Logs** tab. Terminal QR art is often mangled by the log viewer — if you
   cannot scan it, use the **"link locally, then upload the session"** method and
   place `ai_call_session.db` on the persistent disk (a Render Shell on paid plans,
   or a one-off pre-deploy copy step, can do this).

> Note: Render free instances **spin down when idle**. A background worker with no
> HTTP traffic can be suspended, which kills your call bot. A paid instance (or a
> panel host like Katabump) is more reliable for 24/7 calling.

---

### 4️⃣ Railway

Railway is the friendliest of the cloud hosts here because it has volumes and
readable logs.

1. Push the project to GitHub.
2. On Railway: **New Project → Deploy from GitHub repo**.
3. Railway auto-detects Go. If you use Edge TTS you also need Node, so add a
   `Dockerfile` (or a Nixpacks config) that installs both. Otherwise set
   `TTS_ENGINE=geminitts` and plain Go is enough.
   - **Build**: `npm install && go build -o ai-call .`
   - **Start**: `./ai-call`
4. Open the **Variables** tab and add all the keys (`GEMINI_API`, `GROQ_API`,
   `TTS_ENGINE`, `TTS_VOICE`, `TTS_PITCH`, `TTS_SPEED`, `OWNER`, ...).
5. 🔴 **Attach a Volume**: **Service → Settings → Volumes → Add Volume**, and mount
   it at the app's working directory (e.g. `/app`) so `ai_call_session.db` persists
   across redeploys.
6. Deploy, then open **Deployments → View Logs** and **scan the QR code** from the
   log output with *Linked Devices → Link a Device*.
7. Set the restart policy to **Always** so the bot recovers from crashes.

> Railway's trial credits expire; once they run out the service stops and the bot
> goes offline. Keep an eye on usage for a long-running call bot.

---

### 📊 Host comparison

| Host | Interactive console for QR | Persistent session file | 24/7 reliability | Notes |
| :--- | :--- | :--- | :--- | :--- |
| **Katabump** | ✅ Yes (panel console) | ✅ Yes | ⚠️ Free tier limits | Easiest QR scanning |
| **BotHosting** | ✅ Yes (panel console) | ✅ Yes | ⚠️ Free tier limits | Watch the RAM limit |
| **Render** | ❌ Logs only (free) | ⚠️ Needs a Disk | ⚠️ Idle spin-down | Use the upload-session trick |
| **Railway** | ⚠️ Logs (readable) | ⚠️ Needs a Volume | ✅ Good (paid) | Best cloud option |
| **VPS / local** | ✅ Full terminal | ✅ Yes | ✅ Best | Use `screen`/`tmux`/`systemd` |

### Keeping it alive on a plain VPS

```bash
# with screen
screen -S aicall
./ai-call
# detach with Ctrl+A then D

# or as a systemd service
sudo systemctl enable --now ai-call
```

---

## 🧯 Connection Troubleshooting

| Symptom | Cause & fix |
| :--- | :--- |
| QR code appears on **every** restart | `ai_call_session.db` is not persistent. Add a disk/volume, or upload a pre-linked session file. |
| `Failed to connect` on startup | No internet/DNS on the host, or the session was revoked from the phone. Delete `ai_call_session.db*` and re-link. |
| Bot connects but ignores commands | The sender is not in `OWNER`. Use the full international number, no `+`. Or clear `OWNER` to make it public. |
| Incoming calls are rejected instantly | Expected behaviour for non-owners (`call.Reject()` in `OnIncomingCall`). Add the number to `OWNER`. |
| `Failed to place the call` | The target number is not on WhatsApp, is formatted wrongly, or did not answer within the 45-second timeout. |
| Call connects but there is no voice | TTS failed. Check `TTS_ENGINE`; for `edgetts` make sure Node.js and `npm install` are present, or switch to `geminitts`. |
| Bot logged out by itself | You linked the same session in two places at once, or WhatsApp flagged the automation. Re-link and run only one instance. |
| QR is unreadable in the log viewer | Use the "link locally, then upload `ai_call_session.db`" workaround. |

---

## 🎛️ Complete Environment Variables Guide (`.env`)

| Variable | Description | Default / Options |
| :--- | :--- | :--- |
| `GEMINI_API` | Google Gemini API key (required) | API key string |
| `GEMINI_MODEL` | Gemini conversational AI model | `gemini-3.1-flash-lite` |
| `GROQ_API` | Groq Whisper STT API key (required) | API key string |
| `TTS_ENGINE` | TTS engine selection | `edgetts` / `geminitts` / `elevenlabs` / `openai` / `animetts` / `google` |
| `TTS_VOICE` | The voice character to use | `en-US-AvaMultilingualNeural` / `en-US-AndrewMultilingualNeural` / `Puck` / `nova` |
| `TTS_PITCH` | Edge TTS voice pitch adjustment | `-2Hz`, `-1Hz`, `+0Hz`, `+2Hz` |
| `TTS_SPEED` | Audio speaking speed | `1.0` (e.g. `1.2` faster, `0.8` slower) |
| `ELEVENLABS_API` | ElevenLabs API key (optional) | API key string |
| `ELEVEN_VOICE_ID` | ElevenLabs voice ID | `21m00Tcm4TlvDq8ikWAM` (Rachel) |
| `OPENAI_API` | OpenAI API key (optional) | API key string |
| `OWNER` | Whitelist of bot owner phone numbers | `628123456789,628987654321` |
| `SYSTEM_PROMPT` | The AI's speaking-style instructions | A relaxed phone-conversation prompt |

---

## 📖 Full TTS Model & Voice Catalog

To see the complete list of AI models and voice variations across all engines
(Gemini, Edge TTS, ElevenLabs, OpenAI, Anime TTS), read:

👉 **[TTS Models & Voices Catalog (VOICE_MODELS.md)](VOICE_MODELS.md)**

---

## 📱 Interactive Bot Commands

All commands can be sent as WhatsApp text messages (in DMs or groups) with a `!` or `.` prefix:

### 📌 Main commands:
- `!help` / `.help` / `!menu` / `.menu`
  - Shows the full interactive help menu along with the active configuration.
- `!aicall <number>` / `.call <number>`
  - *Example*: `.call 628123456789`
  - The bot automatically calls that number and starts an interactive AI call session.

### ⚙️ Live engine & voice settings (owner only):
- `!engine <engine_name>` / `.engine <engine_name>`
  - *Example*: `.engine edgetts` or `.engine geminitts`
  - Change the TTS engine *live* without restarting the bot!
- `!voice <voice_name>` / `.voice <voice_name>`
  - *Example*: `.voice en-US-AvaMultilingualNeural` or `.voice Puck`
  - Change the voice character *live*!

### 📊 Bot information & status:
- `!status` / `.status` / `!ping` / `.ping`
  - Shows network status, bot uptime, memory (RAM) usage, goroutine count, & the active configuration.
- `!owner` / `.owner`
  - Shows the bot's owner whitelist status.

---

## ⚖️ Disclaimer

This project automates a WhatsApp account through an unofficial protocol client.
Automated calling may violate WhatsApp's Terms of Service and can result in your
number being **banned**. Use a spare number, only call people who have consented
to being called, and use this software at your own risk.

---

## 📜 License
This project is licensed under the [MIT License](LICENSE).
