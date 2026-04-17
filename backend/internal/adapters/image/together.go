package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

const (
	togetherDefaultEndpoint = "https://api.together.xyz/v1/images/generations"
	togetherDefaultModel    = "black-forest-labs/FLUX.1-schnell"
	togetherNoTextSuffix    = "no text, no letters, no words."
)

// Together implements ports.ImageProvider via Together AI FLUX.1-schnell-Free.
// Endpoint: POST https://api.together.xyz/v1/images/generations
// Model:    black-forest-labs/FLUX.1-schnell-Free
type Together struct {
	APIKey   string
	Client   *http.Client
	Log      zerolog.Logger
	Endpoint string
	Model    string
}

func NewTogether(apiKey string, client *http.Client, log zerolog.Logger) *Together {
	if client == nil {
		client = http.DefaultClient
	}
	return &Together{
		APIKey:   apiKey,
		Client:   client,
		Log:      log.With().Str("component", "image.together").Logger(),
		Endpoint: togetherDefaultEndpoint,
		Model:    togetherDefaultModel,
	}
}

type togetherRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Steps          int    `json:"steps"`
	N              int    `json:"n"`
	ResponseFormat string `json:"response_format"`
}

type togetherResponse struct {
	Data []struct {
		URL string `json:"url"`
	} `json:"data"`
}

func (t *Together) Generate(ctx context.Context, req ports.ImageRequest) (ports.ImageResult, error) {
	body := togetherRequest{
		Model:          t.Model,
		Prompt:         buildTogetherPrompt(req),
		Width:          1024,
		Height:         1024,
		Steps:          4,
		N:              1,
		ResponseFormat: "url",
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.together: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.Endpoint, bytes.NewReader(buf))
	if err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.together: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+t.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := t.Client.Do(httpReq)
	if err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.together: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		statusErr := StatusToError(resp.StatusCode)
		return ports.ImageResult{}, fmt.Errorf("image.together: status %d: %w", resp.StatusCode, statusErr)
	}

	var parsed togetherResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.together: decode response: %w", err)
	}
	if len(parsed.Data) == 0 || parsed.Data[0].URL == "" {
		return ports.ImageResult{}, fmt.Errorf("image.together: empty data in response")
	}

	t.Log.Debug().Str("url", parsed.Data[0].URL).Msg("image generated")
	return ports.ImageResult{URL: parsed.Data[0].URL, Provider: "together"}, nil
}

func buildTogetherPrompt(req ports.ImageRequest) string {
	parts := make([]string, 0, 3)
	style := strings.TrimSpace(string(req.StylePrefix))
	if style != "" {
		parts = append(parts, style)
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt != "" {
		parts = append(parts, prompt)
	}
	parts = append(parts, togetherNoTextSuffix)
	return strings.Join(parts, ". ")
}
