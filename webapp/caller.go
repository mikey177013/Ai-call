package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/purpshell/meowcaller"
)

type AICallSession struct {
	Call         *meowcaller.Call
	Conversation *Conversation
	IsActive     bool
	mu           sync.Mutex
	stopChan     chan struct{}
}

func NewAICallSession(call *meowcaller.Call) *AICallSession {
	return &AICallSession{
		Call:         call,
		Conversation: NewConversation(),
		IsActive:     true,
		stopChan:     make(chan struct{}),
	}
}

func (s *AICallSession) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.IsActive {
		s.IsActive = false
		close(s.stopChan)
	}
}

// StartVoiceLoop manages the active AI conversation during a WhatsApp call
func (s *AICallSession) StartVoiceLoop() {
	log.Printf("[AI Call] Voice loop started for call ID: %s (Peer: %s)", s.Call.ID(), s.Call.Peer().String())

	// 1. Play initial welcome greeting
	initialGreeting := "Hello! I am your AI assistant. Please go ahead and speak, I am listening."
	s.speakText(initialGreeting)

	// 2. Main interactive conversation loop
	round := 0
	for {
		select {
		case <-s.stopChan:
			log.Printf("[AI Call] Voice loop stopped for call ID: %s", s.Call.ID())
			return
		default:
		}

		if !s.IsActive || s.Call.State() != meowcaller.CallPhaseActive {
			log.Printf("[AI Call] Call ended or inactive. Stopping voice loop.")
			return
		}

		round++
		recFile := filepath.Join(".", "temp", fmt.Sprintf("rec_%s_%d.wav", s.Call.ID(), round))

		log.Printf("[AI Call] [Round %d] Listening to user speech...", round)

		// Record 6 seconds of user audio
		err := s.recordUserAudio(recFile, 6*time.Second)
		if err != nil {
			log.Printf("[AI Call] Audio recording error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		fi, err := os.Stat(recFile)
		if err != nil || fi.Size() < 4000 {
			_ = os.Remove(recFile)
			continue
		}

		log.Printf("[AI Call] Transcribing audio with Groq STT (whisper-large-v3)...")
		transcription, err := TranscribeAudio(recFile)
		_ = os.Remove(recFile)

		if err != nil {
			log.Printf("[AI Call] STT Error: %v", err)
			continue
		}

		transcription = strings.TrimSpace(transcription)
		if len(transcription) == 0 {
			log.Printf("[AI Call] No speech detected in audio.")
			continue
		}

		log.Printf("[AI Call] User Said: %q", transcription)

		log.Printf("[AI Call] Generating AI response with Gemini (%s)...", AppConfig.GeminiModel)
		aiReply, err := s.Conversation.Chat(transcription)
		if err != nil {
			log.Printf("[AI Call] Gemini AI Error: %v", err)
			s.speakText("Sorry, there was a problem while processing the AI response.")
			continue
		}

		log.Printf("[AI Call] AI Reply: %q", aiReply)

		s.speakText(aiReply)
	}
}

func (s *AICallSession) recordUserAudio(outputPath string, duration time.Duration) error {
	recorder, err := meowcaller.WAVRecorder(outputPath)
	if err != nil {
		return err
	}

	s.Call.Receive(recorder)

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-s.stopChan:
	}

	s.Call.Receive(nil)
	return recorder.Close()
}

func (s *AICallSession) speakText(text string) {
	if text == "" {
		return
	}

	log.Printf("[AI Call] Generating TTS audio for text: %q", text)
	audioFile, err := GenerateTTS(text)
	if err != nil {
		log.Printf("[AI Call] TTS Error: %v", err)
		return
	}
	defer func() {
		time.AfterFunc(2*time.Second, func() {
			_ = os.Remove(audioFile)
		})
	}()

	var src meowcaller.AudioSource
	if filepath.Ext(audioFile) == ".wav" {
		src, err = meowcaller.WAVFile(audioFile)
	} else {
		src, err = meowcaller.MP3File(audioFile)
	}

	if err != nil {
		log.Printf("[AI Call] Failed to open audio source file (%s): %v", audioFile, err)
		return
	}
	defer src.Close()

	doneChan := make(chan struct{})
	player := meowcaller.NewPlayer()
	player.OnFinish(func() {
		close(doneChan)
	})

	s.Call.Subscribe(player)
	player.Play(src)

	select {
	case <-doneChan:
		log.Printf("[AI Call] Finished playing TTS audio.")
	case <-time.After(30 * time.Second):
		log.Printf("[AI Call] Audio playback timeout.")
		player.Stop()
	case <-s.stopChan:
		player.Stop()
	}
}
