package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const HF_BASE_URL = "https://plachta-vits-umamusume-voice-synthesizer.hf.space"

type GradioJoinRequest struct {
	Data        []interface{} `json:"data"`
	EventData   interface{}   `json:"event_data"`
	FnIndex     int           `json:"fn_index"`
	SessionHash string        `json:"session_hash"`
	TriggerID   int           `json:"trigger_id"`
}

type GradioSSEMessage struct {
	Msg    string `json:"msg"`
	Output struct {
		Data []interface{} `json:"data"`
	} `json:"output"`
}

type OpenAITTSRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	Speed          float64 `json:"speed"`
	ResponseFormat string  `json:"response_format"`
}

type ElevenLabsTTSRequest struct {
	Text          string                 `json:"text"`
	ModelID       string                 `json:"model_id"`
	VoiceSettings map[string]interface{} `json:"voice_settings,omitempty"`
}

type GeminiTTSAudioRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig map[string]interface{} `json:"generationConfig"`
}

func randomSessionHash() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateTTS converts text into an audio file (WAV/MP3) path.
// Supports TTS engines: "geminitts", "edgetts", "elevenlabs", "openai", "animetts", and "google" with automatic fallbacks.
func GenerateTTS(text string) (string, error) {
	text = strings.TrimSpace(text)
	if len(text) > 250 {
		cutIdx := strings.IndexAny(text[120:250], ".!?")
		if cutIdx != -1 {
			text = text[:120+cutIdx+1]
		} else {
			text = text[:245] + "..."
		}
	}

	sessionHash := randomSessionHash()
	engine := strings.ToLower(AppConfig.TTSEngine)

	switch engine {
	case "geminitts":
		tempOutPath := filepath.Join(".", "temp", fmt.Sprintf("tts_gemini_%s.wav", sessionHash))
		voice := AppConfig.TTSVoice
		// Gemini TTS only accepts its own prebuilt voice names (Puck, Kore, Aoede, ...).
		// If the configured voice belongs to another engine (Edge "xx-XX-NameNeural" or
		// an OpenAI voice), fall back to the default Gemini voice.
		if voice == "" || isEdgeVoiceName(voice) || isOpenAIVoiceName(voice) {
			voice = "Puck"
		}
		audioPath, err := generateGeminiTTS(text, voice, sessionHash, tempOutPath)
		if err == nil && audioPath != "" {
			return audioPath, nil
		}
		fmt.Printf("[TTS] Gemini TTS failed (%v). Attempting NexRay Gemini TTS API Fallback...\n", err)
		nexrayOutPath := filepath.Join(".", "temp", fmt.Sprintf("tts_nexray_%s.mp3", sessionHash))
		if nPath, nErr := generateNexrayGeminiTTS(text, nexrayOutPath); nErr == nil && nPath != "" {
			return nPath, nil
		}
		fmt.Printf("[TTS] NexRay Gemini TTS failed. Attempting Edge TTS Fallback...\n")
		fallthrough

	case "edgetts":
		tempOutPath := filepath.Join(".", "temp", fmt.Sprintf("tts_edge_%s.mp3", sessionHash))
		voice := AppConfig.TTSVoice
		// Edge TTS requires a "xx-XX-NameNeural" voice name. Anything else (a Gemini
		// voice, an OpenAI voice or an anime character name) is replaced by the default.
		if !isEdgeVoiceName(voice) {
			voice = "en-US-AvaMultilingualNeural"
		}
		audioPath, err := generateEdgeTTS(text, voice, sessionHash, tempOutPath)
		if err == nil && audioPath != "" {
			return audioPath, nil
		}
		fmt.Printf("[TTS] Edge TTS failed (%v). Attempting Google Translate TTS fallback...\n", err)
		fallthrough

	case "elevenlabs":
		tempOutPath := filepath.Join(".", "temp", fmt.Sprintf("tts_eleven_%s.mp3", sessionHash))
		voiceID := AppConfig.ElevenVoiceID
		audioPath, err := generateElevenLabsTTS(text, voiceID, sessionHash, tempOutPath)
		if err == nil && audioPath != "" {
			return audioPath, nil
		}
		fmt.Printf("[TTS] ElevenLabs TTS failed (%v). Attempting Edge TTS / Fallback...\n", err)
		edgePath := filepath.Join(".", "temp", fmt.Sprintf("tts_edge_fb_%s.mp3", sessionHash))
		if aPath, eErr := generateEdgeTTS(text, "en-US-AvaMultilingualNeural", sessionHash, edgePath); eErr == nil {
			return aPath, nil
		}
		fallthrough

	case "openai":
		tempOutPath := filepath.Join(".", "temp", fmt.Sprintf("tts_openai_%s.mp3", sessionHash))
		voice := AppConfig.TTSVoice
		// OpenAI only accepts its own voice names, so replace anything else.
		if !isOpenAIVoiceName(voice) {
			voice = "nova"
		}
		audioPath, err := generateOpenAITTS(text, voice, sessionHash, tempOutPath)
		if err == nil && audioPath != "" {
			return audioPath, nil
		}
		fmt.Printf("[TTS] OpenAI TTS failed (%v). Attempting Edge TTS / Fallback...\n", err)
		edgePath := filepath.Join(".", "temp", fmt.Sprintf("tts_edge_fb_%s.mp3", sessionHash))
		if aPath, eErr := generateEdgeTTS(text, "en-US-AvaMultilingualNeural", sessionHash, edgePath); eErr == nil {
			return aPath, nil
		}
		fallthrough

	case "animetts":
		char := AppConfig.TTSCharacter
		if char == "" {
			// NOTE: the character name is an exact identifier expected by the
			// HuggingFace VITS Gradio API and must be sent verbatim (it includes
			// the original Chinese/Japanese name), so it is not translated.
			char = "特别周 Special Week (Umamusume Pretty Derby)"
		}
		lang := AppConfig.TTSLang
		if lang == "" {
			lang = "Mix"
		}
		tempOutPath := filepath.Join(".", "temp", fmt.Sprintf("tts_%s.wav", sessionHash))
		audioPath, err := generateAnimeTTS(text, char, lang, sessionHash, tempOutPath)
		if err == nil && audioPath != "" {
			return audioPath, nil
		}
		fmt.Printf("[TTS] Anime TTS failed/timed out (%v). Using Edge TTS fallback...\n", err)
		edgePath := filepath.Join(".", "temp", fmt.Sprintf("tts_edge_fb_%s.mp3", sessionHash))
		if aPath, eErr := generateEdgeTTS(text, "en-US-AvaMultilingualNeural", sessionHash, edgePath); eErr == nil {
			return aPath, nil
		}
		fallthrough

	case "google":
		tempOutPath := filepath.Join(".", "temp", fmt.Sprintf("tts_google_%s.mp3", sessionHash))
		return generateGoogleTTS(text, tempOutPath)

	default:
		tempOutPath := filepath.Join(".", "temp", fmt.Sprintf("tts_edge_%s.mp3", sessionHash))
		if aPath, err := generateEdgeTTS(text, "en-US-AvaMultilingualNeural", sessionHash, tempOutPath); err == nil {
			return aPath, nil
		}
		return generateGoogleTTS(text, filepath.Join(".", "temp", fmt.Sprintf("tts_google_%s.mp3", sessionHash)))
	}
}

func generateGeminiTTS(text, voice, sessionHash, outputPath string) (string, error) {
	apiKey := AppConfig.GeminiAPI
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API key is not configured")
	}

	if voice == "" {
		voice = "Puck"
	}

	reqBody := GeminiTTSAudioRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: fmt.Sprintf("Say the following in English in a friendly, expressive and natural tone: %s", text)},
				},
			},
		},
		GenerationConfig: map[string]interface{}{
			"responseModalities": []string{"AUDIO"},
			"speechConfig": map[string]interface{}{
				"voiceConfig": map[string]interface{}{
					"prebuiltVoiceConfig": map[string]interface{}{
						"voiceName": voice,
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-preview-tts:generateContent?key=%s", apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini TTS API returned status %d: %s", resp.StatusCode, string(b))
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var geminiAudioResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					InlineData struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBytes, &geminiAudioResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal gemini audio response: %w", err)
	}

	if len(geminiAudioResp.Candidates) == 0 || len(geminiAudioResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty audio data candidate from Gemini TTS")
	}

	base64Data := geminiAudioResp.Candidates[0].Content.Parts[0].InlineData.Data
	if base64Data == "" {
		return "", fmt.Errorf("empty base64 audio data from Gemini TTS")
	}

	rawPCM, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 pcm audio: %w", err)
	}

	wavHeader := createWavHeader(len(rawPCM), 24000, 1, 16)
	out, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := out.Write(wavHeader); err != nil {
		return "", err
	}
	if _, err := out.Write(rawPCM); err != nil {
		return "", err
	}

	return outputPath, nil
}

func createWavHeader(dataLength, sampleRate, numChannels, bitsPerSample int) []byte {
	header := make([]byte, 44)
	copy(header[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataLength))
	copy(header[8:12], []byte("WAVE"))
	copy(header[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1) // PCM format
	binary.LittleEndian.PutUint16(header[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	byteRate := sampleRate * numChannels * (bitsPerSample / 8)
	binary.LittleEndian.PutUint32(header[28:32], uint32(byteRate))
	blockAlign := numChannels * (bitsPerSample / 8)
	binary.LittleEndian.PutUint16(header[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitsPerSample))
	copy(header[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataLength))
	return header
}

// isEdgeVoiceName reports whether the given voice looks like a Microsoft Edge
// Neural voice name, i.e. "<lang>-<REGION>-<Name>Neural" (e.g. "en-US-AvaMultilingualNeural").
func isEdgeVoiceName(voice string) bool {
	if !strings.HasSuffix(voice, "Neural") {
		return false
	}
	return len(strings.Split(voice, "-")) >= 3
}

// isOpenAIVoiceName reports whether the given voice is one of the OpenAI Audio TTS voices.
func isOpenAIVoiceName(voice string) bool {
	switch strings.ToLower(voice) {
	case "alloy", "ash", "coral", "echo", "fable", "nova", "onyx", "sage", "shimmer":
		return true
	}
	return false
}

// edgeVoiceLocale extracts the "<lang>-<REGION>" locale prefix from an Edge voice
// name so the TTS request is tagged with the language that matches the voice.
func edgeVoiceLocale(voice string) string {
	parts := strings.Split(voice, "-")
	if len(parts) >= 3 {
		return parts[0] + "-" + parts[1]
	}
	return "en-US"
}

func generateEdgeTTS(text, voice, sessionHash, outputPath string) (string, error) {
	if voice == "" {
		voice = "en-US-AvaMultilingualNeural"
	}

	lang := edgeVoiceLocale(voice)

	speed := AppConfig.TTSSpeed
	rateStr := fmt.Sprintf("%+d%%", int((speed-1.0)*100))

	pitch := AppConfig.TTSPitch
	if pitch == "" {
		pitch = "+0Hz"
	}

	cmd := exec.Command("node", "-e", fmt.Sprintf(`
const { EdgeTTS } = require("node-edge-tts");
(async () => {
  try {
    const tts = new EdgeTTS({ voice: %q, lang: %q, outputFormat: "audio-24khz-48kbitrate-mono-mp3", rate: %q, pitch: %q });
    await tts.ttsPromise(%q, %q);
    process.exit(0);
  } catch(e) {
    console.error(e);
    process.exit(1);
  }
})();
`, voice, lang, rateStr, pitch, text, outputPath))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("edge TTS node execution failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}

	fi, err := os.Stat(outputPath)
	if err != nil || fi.Size() == 0 {
		return "", fmt.Errorf("edge TTS generated empty audio file")
	}

	return outputPath, nil
}

func generateElevenLabsTTS(text, voiceID, sessionHash, outputPath string) (string, error) {
	apiKey := AppConfig.ElevenAPI
	if apiKey == "" {
		return "", fmt.Errorf("ELEVENLABS_API key is not configured")
	}

	if voiceID == "" {
		voiceID = "21m00Tcm4TlvDq8ikWAM" // Rachel
	}

	reqBody := ElevenLabsTTSRequest{
		Text:    text,
		ModelID: "eleven_multilingual_v2",
		VoiceSettings: map[string]interface{}{
			"stability":        0.5,
			"similarity_boost": 0.75,
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voiceID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("elevenlabs TTS returned status %d: %s", resp.StatusCode, string(b))
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func generateOpenAITTS(text, voice, sessionHash, outputPath string) (string, error) {
	apiKey := AppConfig.OpenAIAPI
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API key is not configured")
	}

	if voice == "" {
		voice = "nova"
	}

	reqBody := OpenAITTSRequest{
		Model:          "tts-1",
		Input:          text,
		Voice:          voice,
		Speed:          AppConfig.TTSSpeed,
		ResponseFormat: "mp3",
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/speech", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai TTS returned status %d: %s", resp.StatusCode, string(b))
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return outputPath, nil
}

func generateAnimeTTS(text, char, lang, sessionHash, outputPath string) (string, error) {
	client := &http.Client{Timeout: 45 * time.Second}

	joinReq := GradioJoinRequest{
		Data: []interface{}{
			text,
			char,
			lang,
			AppConfig.TTSSpeed, // speed
			false,              // noise
		},
		EventData:   nil,
		FnIndex:     2,
		SessionHash: sessionHash,
		TriggerID:   24,
	}

	jsonBytes, err := json.Marshal(joinReq)
	if err != nil {
		return "", err
	}

	resp, err := client.Post(HF_BASE_URL+"/gradio_api/queue/join", "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("queue join failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("queue join returned status %d: %s", resp.StatusCode, string(b))
	}

	pollURL := fmt.Sprintf("%s/gradio_api/queue/data?session_hash=%s", HF_BASE_URL, sessionHash)
	deadline := time.Now().Add(35 * time.Second)

	for time.Now().Before(deadline) {
		time.Sleep(1500 * time.Millisecond)

		req, _ := http.NewRequest("GET", pollURL, nil)
		pollResp, err := client.Do(req)
		if err != nil {
			continue
		}

		bodyBytes, err := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()
		if err != nil {
			continue
		}

		lines := strings.Split(string(bodyBytes), "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "data:") {
				continue
			}

			jsonData := strings.TrimPrefix(line, "data:")
			jsonData = strings.TrimSpace(jsonData)

			var msg GradioSSEMessage
			if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
				continue
			}

			if msg.Msg == "process_completed" {
				for _, item := range msg.Output.Data {
					var fileURL string

					switch v := item.(type) {
					case map[string]interface{}:
						if urlVal, ok := v["url"].(string); ok && urlVal != "" {
							fileURL = urlVal
						} else if nameVal, ok := v["name"].(string); ok && nameVal != "" {
							fileURL = HF_BASE_URL + "/file=" + nameVal
						}
					case string:
						if strings.HasPrefix(v, "http") || strings.HasPrefix(v, "/file=") {
							fileURL = v
						}
					}

					if fileURL != "" {
						if !strings.HasPrefix(fileURL, "http") {
							fileURL = HF_BASE_URL + fileURL
						}
						err := downloadFile(fileURL, outputPath)
						if err == nil {
							return outputPath, nil
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("Anime TTS processing timeout")
}

func generateGoogleTTS(text, outputPath string) (string, error) {
	encodedText := url.QueryEscape(text)
	ttsURL := fmt.Sprintf("https://translate.google.com/translate_tts?ie=UTF-8&tl=en&client=tw-ob&q=%s", encodedText)

	err := downloadFile(ttsURL, outputPath)
	if err != nil {
		return "", err
	}
	return outputPath, nil
}

func downloadFile(fileURL, outputPath string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", fileURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

type NexrayTTSResponse struct {
	Status bool   `json:"status"`
	Result string `json:"result"`
}

func generateNexrayGeminiTTS(text, outputPath string) (string, error) {
	encodedText := url.QueryEscape(text)
	apiURL := fmt.Sprintf("https://api.nexray.eu.cc/ai/gemini-tts?text=%s", encodedText)

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("nexray API HTTP GET error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("nexray API returned status %d: %s", resp.StatusCode, string(b))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read nexray API response body: %w", err)
	}

	var nexrayResp NexrayTTSResponse
	if err := json.Unmarshal(bodyBytes, &nexrayResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal nexray response: %w", err)
	}

	if !nexrayResp.Status || nexrayResp.Result == "" {
		return "", fmt.Errorf("nexray API returned status false or empty audio url")
	}

	if err := downloadFile(nexrayResp.Result, outputPath); err != nil {
		return "", fmt.Errorf("failed to download audio from nexray url (%s): %w", nexrayResp.Result, err)
	}

	return outputPath, nil
}
