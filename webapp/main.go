package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/purpshell/meowcaller"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

var botStartTime = time.Now()

func main() {
	log.Println("==================================================")
	log.Println("     WhatsApp AI Call Assistant Bot (Golang)      ")
	log.Println("==================================================")

	// Load configuration from .env
	cfg := LoadConfig()
	log.Printf("[Config] GROQ_API status: %t", cfg.GroqAPI != "")
	log.Printf("[Config] GEMINI_API status: %t", cfg.GeminiAPI != "")
	log.Printf("[Config] GEMINI_MODEL: %s", cfg.GeminiModel)
	log.Printf("[Config] TTS Character: %s", cfg.TTSCharacter)
	log.Printf("[Config] TTS Speed: %.1f", cfg.TTSSpeed)
	if len(cfg.Owners) > 0 {
		log.Printf("[Config] Owners configured: %v", cfg.Owners)
	} else {
		log.Println("[Config] Owners: Public (Anyone can call)")
	}

	// Setup SQLite container store for WhatsApp session
	dbLog := waLog.Stdout("Database", "WARN", true)
	container, err := sqlstore.New(context.Background(), "sqlite", "file:ai_call_session.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", dbLog)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		log.Fatalf("Failed to get device store: %v", err)
	}

	// Create whatsmeow client
	clientLog := waLog.Stdout("Client", "INFO", true)
	waClient := whatsmeow.NewClient(deviceStore, clientLog)

	// Initialize meowcaller client
	callerClient := meowcaller.NewClient(waClient)

	// Setup Incoming Call Handler
	callerClient.OnIncomingCall(func(call *meowcaller.Call) {
		peer := call.Peer().User
		log.Printf("[Call] Incoming call from %s (Call ID: %s)", call.Peer().String(), call.ID())

		if !IsOwner(peer) {
			log.Printf("[Call] Rejecting incoming call from non-owner: %s", peer)
			_ = call.Reject()
			return
		}

		// Answer incoming call
		if err := call.Answer(); err != nil {
			log.Printf("[Call] Failed to answer call: %v", err)
			return
		}

		session := NewAICallSession(call)
		call.OnEnd(func(reason string) {
			log.Printf("[Call] Call ended (Reason: %s)", reason)
			session.Stop()
		})

		// Start AI Voice Loop in background
		go session.StartVoiceLoop()
	})

	// Setup Message Handler for commands (e.g., !aicall 628xxx)
	waClient.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			handleMessage(waClient, callerClient, v)
		}
	})

	// QR Code / Session Login
	if waClient.Store.ID == nil {
		qrChan, _ := waClient.GetQRChannel(context.Background())
		err = waClient.Connect()
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("\nScan the QR code below using WhatsApp on your phone:")
				fmt.Println("WhatsApp > Settings > Linked Devices > Link a Device")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Printf("QR Event: %s\n", evt.Event)
			}
		}
	} else {
		err = waClient.Connect()
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
		log.Println("Successfully connected to WhatsApp!")
	}

	// Keep program running until interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down AI Call Assistant...")
	waClient.Disconnect()
}

func handleMessage(client *whatsmeow.Client, callerClient *meowcaller.Client, msg *events.Message) {
	if msg.Info.IsFromMe {
		return
	}

	body := getMessageBody(msg.Message)
	if body == "" {
		return
	}

	fields := strings.Fields(body)
	if len(fields) == 0 {
		return
	}
	cmd := strings.ToLower(fields[0])

	switch cmd {
	case "!help", ".help", "!menu", ".menu":
		helpText := fmt.Sprintf(`🤖 *WHATSAPP AI CALL ASSISTANT BOT* 🤖

📌 *Main Commands:*
• *!aicall <number>* / *.call <number>*
  ↳ Place an AI phone call to the target number.

⚙️ *Live Engine Settings (Owner Only):*
• *!engine <engine_name>*
  ↳ Switch the TTS engine live (edgetts / geminitts / elevenlabs / openai / animetts).
• *!voice <voice_name>*
  ↳ Switch the TTS voice live (e.g. en-US-AvaMultilingualNeural, en-US-AndrewMultilingualNeural, Puck, nova).

📊 *Bot Information & Status:*
• *!status* / *.ping*
  ↳ Check system status, uptime, memory, & bot details.
• *!owner* / *.owner*
  ↳ Check the bot owner whitelist status.
• *!help* / *.menu*
  ↳ Show this help menu.

-----------------------------------
ℹ️ *Active Configuration:*
• AI Model  : %s
• TTS Engine: %s
• TTS Voice : %s
• Pitch     : %s`, AppConfig.GeminiModel, AppConfig.TTSEngine, AppConfig.TTSVoice, AppConfig.TTSPitch)
		sendReply(client, msg, helpText)

	case "!status", ".status", "!ping", ".ping":
		uptime := time.Since(botStartTime).Truncate(time.Second)
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		allocMB := float64(memStats.Alloc) / 1024 / 1024

		statusText := fmt.Sprintf(`⚡ *BOT STATUS & SYSTEM METRICS* ⚡

• *Status Connection*: Connected (Online)
• *Bot Uptime*      : %s
• *Memory Usage*    : %.2f MB
• *Goroutines*      : %d
• *AI Model*        : %s
• *TTS Engine*      : %s
• *TTS Voice*       : %s
• *Owner Restricted*: %t`, uptime, allocMB, runtime.NumGoroutine(), AppConfig.GeminiModel, AppConfig.TTSEngine, AppConfig.TTSVoice, len(AppConfig.Owners) > 0)
		sendReply(client, msg, statusText)

	case "!owner", ".owner":
		var ownerText string
		if len(AppConfig.Owners) > 0 {
			ownerText = fmt.Sprintf("👑 *BOT OWNER LIST:*\n• %s\n\n_Only the numbers listed above may use the AI call commands._", strings.Join(AppConfig.Owners, "\n• "))
		} else {
			ownerText = "🔓 *BOT STATUS:* Public (anyone can place an AI call)."
		}
		sendReply(client, msg, ownerText)

	case "!engine", ".engine":
		senderUser := msg.Info.Sender.User
		if !IsOwner(senderUser) {
			sendReply(client, msg, "Sorry, this command can only be used by the bot owner.")
			return
		}
		if len(fields) < 2 {
			sendReply(client, msg, "Usage: !engine <edgetts|geminitts|elevenlabs|openai|animetts>")
			return
		}
		newEngine := strings.ToLower(fields[1])
		switch newEngine {
		case "edgetts", "geminitts", "elevenlabs", "openai", "animetts", "google":
			AppConfig.TTSEngine = newEngine
			if newEngine == "geminitts" {
				AppConfig.TTSVoice = "Puck"
			} else if newEngine == "edgetts" {
				AppConfig.TTSVoice = "en-US-AvaMultilingualNeural"
			}
			sendReply(client, msg, fmt.Sprintf("✅ TTS engine successfully changed to: *%s* (default voice: %s)", newEngine, AppConfig.TTSVoice))
		default:
			sendReply(client, msg, "Invalid engine. Available options: edgetts, geminitts, elevenlabs, openai, animetts, google.")
		}

	case "!voice", ".voice":
		senderUser := msg.Info.Sender.User
		if !IsOwner(senderUser) {
			sendReply(client, msg, "Sorry, this command can only be used by the bot owner.")
			return
		}
		if len(fields) < 2 {
			sendReply(client, msg, "Usage: !voice <voice_name>\nExample: !voice en-US-AvaMultilingualNeural")
			return
		}
		newVoice := fields[1]
		AppConfig.TTSVoice = newVoice
		sendReply(client, msg, fmt.Sprintf("✅ TTS voice successfully changed to: *%s*", newVoice))

	case "!aicall", ".aicall", "!call", ".call":
		senderUser := msg.Info.Sender.User
		if !IsOwner(senderUser) {
			log.Printf("[Command] Denied call command from non-owner: %s", senderUser)
			sendReply(client, msg, "Sorry, this feature can only be used by the bot owner.")
			return
		}

		if len(fields) < 2 {
			sendReply(client, msg, "Usage: !aicall 628xxxxxxxx")
			return
		}

		targetNum := fields[1]
		targetNum = strings.TrimPrefix(targetNum, "+")
		targetNum = strings.ReplaceAll(targetNum, "-", "")
		targetNum = strings.ReplaceAll(targetNum, " ", "")

		if targetNum == "" {
			sendReply(client, msg, "Usage: !aicall 628xxxxxxxx")
			return
		}

		targetJID := types.NewJID(targetNum, types.DefaultUserServer)
		sendReply(client, msg, fmt.Sprintf("Starting an AI call to %s...", targetNum))

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		log.Printf("[Command] Dialing %s for AI call...", targetJID.String())
		call, err := callerClient.Call(ctx, targetJID.String())
		if err != nil {
			log.Printf("[Command] Failed to place call: %v", err)
			sendReply(client, msg, fmt.Sprintf("Failed to place the call: %v", err))
			return
		}

		session := NewAICallSession(call)
		call.OnEnd(func(reason string) {
			log.Printf("[Call] Call with %s ended: %s", targetJID.String(), reason)
			session.Stop()
		})

		call.OnReady(func() {
			log.Printf("[Call] Call connected with %s! Starting AI voice loop...", targetJID.String())
			go session.StartVoiceLoop()
		})
	}
}

func sendReply(client *whatsmeow.Client, msg *events.Message, text string) {
	participant := msg.Info.Sender.String()
	stanzaID := msg.Info.ID

	_, _ = client.SendMessage(context.Background(), msg.Info.Chat, &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: buildStringPointer(text),
			ContextInfo: &waProto.ContextInfo{
				StanzaID:      buildStringPointer(stanzaID),
				Participant:   buildStringPointer(participant),
				QuotedMessage: msg.Message,
			},
		},
	})
}

func getMessageBody(m *waProto.Message) string {
	if m == nil {
		return ""
	}
	if m.GetEphemeralMessage() != nil {
		return getMessageBody(m.GetEphemeralMessage().GetMessage())
	}
	if m.GetViewOnceMessage() != nil {
		return getMessageBody(m.GetViewOnceMessage().GetMessage())
	}
	if m.GetViewOnceMessageV2() != nil {
		return getMessageBody(m.GetViewOnceMessageV2().GetMessage())
	}
	if m.GetDocumentWithCaptionMessage() != nil {
		return getMessageBody(m.GetDocumentWithCaptionMessage().GetMessage())
	}
	if m.GetConversation() != "" {
		return m.GetConversation()
	}
	if m.GetExtendedTextMessage() != nil {
		return m.GetExtendedTextMessage().GetText()
	}
	return ""
}

func buildStringPointer(s string) *string {
	return &s
}
