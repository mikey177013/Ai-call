package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type GroqTranscriptionResponse struct {
	Text  string `json:"text"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// TranscribeAudio sends an audio file (WAV/MP3/OGG) to Groq STT (whisper-large-v3)
func TranscribeAudio(audioPath string) (string, error) {
	apiKey := AppConfig.GroqAPI
	if apiKey == "" {
		return "", fmt.Errorf("GROQ_API key is not configured")
	}

	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy audio file data: %w", err)
	}

	if err := writer.WriteField("model", "whisper-large-v3"); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}

	_ = writer.WriteField("response_format", "json")
	_ = writer.WriteField("temperature", "0.0")

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/audio/transcriptions", body)
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var result GroqTranscriptionResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("failed to parse groq response json: %w", err)
	}

	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("groq API error: %s", result.Error.Message)
	}

	return result.Text, nil
}
