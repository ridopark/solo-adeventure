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

const falDefaultEndpoint = "https://fal.run/fal-ai/flux/schnell"

type Fal struct {
	APIKey   string
	Endpoint string
	Client   *http.Client
	Log      zerolog.Logger
}

func NewFal(apiKey string, client *http.Client, log zerolog.Logger) *Fal {
	if client == nil {
		client = http.DefaultClient
	}
	return &Fal{
		APIKey:   apiKey,
		Endpoint: falDefaultEndpoint,
		Client:   client,
		Log:      log.With().Str("component", "image.fal").Logger(),
	}
}

type falRequest struct {
	Prompt              string `json:"prompt"`
	ImageSize           string `json:"image_size"`
	NumInferenceSteps   int    `json:"num_inference_steps"`
	NumImages           int    `json:"num_images"`
	EnableSafetyChecker bool   `json:"enable_safety_checker"`
}

type falImage struct {
	URL string `json:"url"`
}

type falResponse struct {
	Images []falImage `json:"images"`
}

func (f *Fal) Generate(ctx context.Context, req ports.ImageRequest) (ports.ImageResult, error) {
	prompt := buildFalPrompt(string(req.StylePrefix), req.Prompt)

	payload := falRequest{
		Prompt:              prompt,
		ImageSize:           "square_hd",
		NumInferenceSteps:   4,
		NumImages:           1,
		EnableSafetyChecker: true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.fal: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, f.Endpoint, bytes.NewReader(body))
	if err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.fal: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Key "+f.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := f.Client.Do(httpReq)
	if err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.fal: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		classified := StatusToError(resp.StatusCode)
		return ports.ImageResult{}, fmt.Errorf("image.fal: status %d: %w", resp.StatusCode, classified)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.fal: read response: %w", err)
	}

	var decoded falResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.fal: decode response: %w", err)
	}
	if len(decoded.Images) == 0 || decoded.Images[0].URL == "" {
		return ports.ImageResult{}, fmt.Errorf("image.fal: empty images in response")
	}

	return ports.ImageResult{URL: decoded.Images[0].URL, Provider: "fal"}, nil
}

func buildFalPrompt(stylePrefix, userPrompt string) string {
	const suffix = "no text, no letters, no words."
	parts := make([]string, 0, 3)
	if s := strings.TrimSpace(stylePrefix); s != "" {
		parts = append(parts, s)
	}
	if p := strings.TrimSpace(userPrompt); p != "" {
		parts = append(parts, p)
	}
	parts = append(parts, suffix)
	return strings.Join(parts, ". ")
}
