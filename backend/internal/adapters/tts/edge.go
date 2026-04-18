package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

type Edge struct {
	baseURL string
	voice   string
	client  *http.Client
	log     zerolog.Logger
}

func NewEdge(baseURL, voice string, log zerolog.Logger) *Edge {
	return &Edge{
		baseURL: baseURL,
		voice:   voice,
		client:  &http.Client{Timeout: 120 * time.Second},
		log:     log.With().Str("component", "tts.edge").Logger(),
	}
}

type edgeReqBody struct {
	Text  string `json:"text"`
	Voice string `json:"voice,omitempty"`
}

func (e *Edge) Synthesize(ctx context.Context, req ports.TTSRequest) (ports.TTSResult, error) {
	voice := req.Voice
	if voice == "" {
		voice = e.voice
	}
	body, err := json.Marshal(edgeReqBody{Text: req.Text, Voice: voice})
	if err != nil {
		return ports.TTSResult{}, fmt.Errorf("tts: marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/tts", bytes.NewReader(body))
	if err != nil {
		return ports.TTSResult{}, fmt.Errorf("tts: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(httpReq)
	if err != nil {
		return ports.TTSResult{}, fmt.Errorf("tts: sidecar unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return ports.TTSResult{}, fmt.Errorf("tts: sidecar status %d: %s", resp.StatusCode, string(snippet))
	}
	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return ports.TTSResult{}, fmt.Errorf("tts: read audio: %w", err)
	}
	if len(audio) == 0 {
		return ports.TTSResult{}, errors.New("tts: empty audio payload")
	}
	return ports.TTSResult{Audio: audio, Format: "audio/mpeg"}, nil
}
