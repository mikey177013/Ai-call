package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiContent struct {
	Role  string       `json:"role"` // "user" or "model"
	Parts []GeminiPart `json:"parts"`
}

type GeminiSystemInstruction struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiRequest struct {
	Contents          []GeminiContent          `json:"contents"`
	SystemInstruction *GeminiSystemInstruction `json:"systemInstruction,omitempty"`
	GenerationConfig  map[string]interface{}   `json:"generationConfig,omitempty"`
}

type GeminiCandidate struct {
	Content GeminiContent `json:"content"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type Conversation struct {
	History []GeminiContent
}

func NewConversation() *Conversation {
	return &Conversation{
		History: make([]GeminiContent, 0),
	}
}

// Chat sends prompt to Gemini AI model and returns response text
func (c *Conversation) Chat(userPrompt string) (string, error) {
	apiKey := AppConfig.GeminiAPI
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API key is not configured")
	}

	c.History = append(c.History, GeminiContent{
		Role:  "user",
		Parts: []GeminiPart{{Text: userPrompt}},
	})

	sysPrompt := AppConfig.SystemPrompt
	if sysPrompt == "" {
		sysPrompt = DefaultSystemPrompt
	}

	reqBody := GeminiRequest{
		Contents: c.History,
		SystemInstruction: &GeminiSystemInstruction{
			Parts: []GeminiPart{{Text: sysPrompt}},
		},
		GenerationConfig: map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 100,
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	primaryModel := AppConfig.GeminiModel
	if primaryModel == "" {
		primaryModel = "gemini-3.1-flash-lite"
	}

	fallbackModels := []string{primaryModel, "gemini-3.1-flash-lite", "gemini-3.5-flash", "gemini-flash-latest"}
	// Deduplicate
	modelsToTry := make([]string, 0, len(fallbackModels))
	seen := make(map[string]bool)
	for _, m := range fallbackModels {
		if !seen[m] {
			seen[m] = true
			modelsToTry = append(modelsToTry, m)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var lastErr error
	var respBytes []byte

	for _, modelName := range modelsToTry {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelName, apiKey)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusOK {
			respBytes = b
			lastErr = nil
			break
		}

		lastErr = fmt.Errorf("gemini API (%s) returned status %d: %s", modelName, resp.StatusCode, string(b))
	}

	if lastErr != nil {
		return "", lastErr
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal gemini response: %w", err)
	}

	if geminiResp.Error != nil && geminiResp.Error.Message != "" {
		return "", fmt.Errorf("gemini API error: %s", geminiResp.Error.Message)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response candidate from Gemini")
	}

	reply := geminiResp.Candidates[0].Content.Parts[0].Text

	// Append assistant response to history
	c.History = append(c.History, GeminiContent{
		Role:  "model",
		Parts: []GeminiPart{{Text: reply}},
	})

	return reply, nil
}
