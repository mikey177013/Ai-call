package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	GroqAPI       string
	GeminiAPI     string
	GeminiModel   string
	OpenAIAPI     string
	ElevenAPI     string
	SystemPrompt  string
	PromptLang    string
	TTSEngine     string // "geminitts", "edgetts", "elevenlabs", "openai", "animetts", "google"
	TTSCharacter  string
	TTSVoice      string // Voice for Gemini TTS ("Puck", "Kore", "Aoede"), Edge TTS ("ms-MY-YasminNeural"), OpenAI ("nova")
	ElevenVoiceID string // ElevenLabs Voice ID
	TTSLang       string
	TTSSpeed      float64
	TTSPitch      string // Edge TTS Pitch (e.g. "+0Hz", "-1Hz")
	Owners        []string
}

var AppConfig *Config

const DefaultSystemPrompt = `You are a smart AI voice assistant speaking to someone over a WhatsApp phone call.
Important rules:
1. Answer SHORTLY, in a FRIENDLY and NATURAL way (maximum 1-2 short sentences, maximum 25 words).
2. DO NOT give answers that are too long, rambling, or formatted as a list, so the phone audio does not get cut off.
3. DO NOT use markdown formatting such as asterisks (*), hashes (#), bullet points (-), or table symbols.
4. Use casual and polite English.`

func LoadConfig() *Config {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	groqAPI := os.Getenv("GROQ_API")
	if groqAPI == "" {
		groqAPI = os.Getenv("GROQ_API_KEY")
	}

	geminiAPI := os.Getenv("GEMINI_API")
	if geminiAPI == "" {
		geminiAPI = os.Getenv("GEMINI_API_KEY")
	}

	openAIAPI := os.Getenv("OPENAI_API")
	if openAIAPI == "" {
		openAIAPI = os.Getenv("OPENAI_API_KEY")
	}

	elevenAPI := os.Getenv("ELEVENLABS_API")
	if elevenAPI == "" {
		elevenAPI = os.Getenv("ELEVEN_API")
	}
	if elevenAPI == "" {
		elevenAPI = os.Getenv("ELEVENLABS_API_KEY")
	}

	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-3.1-flash-lite"
	}

	sysPrompt := os.Getenv("SYSTEM_PROMPT")
	if sysPrompt == "" {
		sysPrompt = DefaultSystemPrompt
	}

	promptLang := os.Getenv("PROMPT_LANG")
	if promptLang == "" {
		promptLang = "en"
	}

	ttsEngine := strings.ToLower(strings.TrimSpace(os.Getenv("TTS_ENGINE")))
	if ttsEngine == "" {
		if elevenAPI != "" {
			ttsEngine = "elevenlabs"
		} else if openAIAPI != "" {
			ttsEngine = "openai"
		} else {
			ttsEngine = "edgetts"
		}
	}

	ttsChar := os.Getenv("TTS_CHARACTER")
	if ttsChar == "" {
		// NOTE: Anime TTS character names are exact identifiers required by the
		// HuggingFace VITS Gradio API and must be kept verbatim (not translated).
		ttsChar = "特别周 Special Week (Umamusume Pretty Derby)"
	}

	ttsVoice := os.Getenv("TTS_VOICE")
	if ttsVoice == "" {
		if ttsEngine == "edgetts" {
			ttsVoice = "en-US-AvaMultilingualNeural"
		} else if ttsEngine == "geminitts" {
			ttsVoice = "Puck"
		} else {
			ttsVoice = "nova"
		}
	}

	elevenVoiceID := os.Getenv("ELEVEN_VOICE_ID")
	if elevenVoiceID == "" {
		elevenVoiceID = os.Getenv("ELEVENLABS_VOICE_ID")
	}
	if elevenVoiceID == "" {
		elevenVoiceID = "21m00Tcm4TlvDq8ikWAM" // Rachel
	}

	ttsLang := os.Getenv("TTS_LANG")
	if ttsLang == "" {
		ttsLang = "Mix"
	}

	ttsSpeedStr := os.Getenv("TTS_SPEED")
	ttsSpeed := 1.0
	if ttsSpeedStr != "" {
		if val, err := strconv.ParseFloat(ttsSpeedStr, 64); err == nil && val > 0 {
			ttsSpeed = val
		}
	}

	ttsPitch := os.Getenv("TTS_PITCH")
	if ttsPitch == "" {
		ttsPitch = "-1Hz"
	}

	ownerStr := os.Getenv("OWNER")
	var owners []string
	if ownerStr != "" {
		parts := strings.Split(ownerStr, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				owners = append(owners, p)
			}
		}
	}

	AppConfig = &Config{
		GroqAPI:       groqAPI,
		GeminiAPI:     geminiAPI,
		GeminiModel:   geminiModel,
		OpenAIAPI:     openAIAPI,
		ElevenAPI:     elevenAPI,
		SystemPrompt:  sysPrompt,
		PromptLang:    promptLang,
		TTSEngine:     ttsEngine,
		TTSCharacter:  ttsChar,
		TTSVoice:      ttsVoice,
		ElevenVoiceID: elevenVoiceID,
		TTSLang:       ttsLang,
		TTSSpeed:      ttsSpeed,
		TTSPitch:      ttsPitch,
		Owners:        owners,
	}

	if AppConfig.GroqAPI == "" {
		log.Println("[WARNING] GROQ_API key is missing in .env! STT functionality might fail.")
	}
	if AppConfig.GeminiAPI == "" {
		log.Println("[WARNING] GEMINI_API key is missing in .env! AI chat functionality might fail.")
	}

	tempDir := filepath.Join(".", "temp")
	_ = os.MkdirAll(tempDir, 0755)

	return AppConfig
}

func IsOwner(sender string) bool {
	if len(AppConfig.Owners) == 0 {
		return true
	}
	cleanedSender := strings.TrimPrefix(sender, "+")
	cleanedSender = strings.Split(cleanedSender, "@")[0]
	cleanedSender = strings.Split(cleanedSender, ".")[0]
	cleanedSender = strings.TrimSpace(cleanedSender)

	for _, owner := range AppConfig.Owners {
		cleanedOwner := strings.TrimPrefix(owner, "+")
		cleanedOwner = strings.Split(cleanedOwner, "@")[0]
		cleanedOwner = strings.TrimSpace(cleanedOwner)
		if cleanedOwner == "" {
			continue
		}
		if cleanedSender == cleanedOwner || strings.HasSuffix(cleanedSender, cleanedOwner) || strings.HasPrefix(cleanedSender, cleanedOwner) {
			return true
		}
	}
	return false
}
