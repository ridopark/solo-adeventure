package depth

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

type Local struct {
	baseURL string
	client  *http.Client
	log     zerolog.Logger
}

func NewLocal(baseURL string, log zerolog.Logger) *Local {
	return &Local{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
		log:     log.With().Str("component", "depth.local").Logger(),
	}
}

type depthReqBody struct {
	ImageURL string `json:"image_url"`
}

func (l *Local) Estimate(ctx context.Context, req ports.DepthRequest) (ports.DepthResult, error) {
	body, err := json.Marshal(depthReqBody{ImageURL: req.ImageURL})
	if err != nil {
		return ports.DepthResult{}, fmt.Errorf("depth: marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, l.baseURL+"/depth", bytes.NewReader(body))
	if err != nil {
		return ports.DepthResult{}, fmt.Errorf("depth: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := l.client.Do(httpReq)
	if err != nil {
		return ports.DepthResult{}, fmt.Errorf("depth: sidecar unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return ports.DepthResult{}, fmt.Errorf("depth: sidecar status %d: %s", resp.StatusCode, string(snippet))
	}
	png, err := io.ReadAll(resp.Body)
	if err != nil {
		return ports.DepthResult{}, fmt.Errorf("depth: read png: %w", err)
	}
	if len(png) == 0 {
		return ports.DepthResult{}, errors.New("depth: empty png payload")
	}
	return ports.DepthResult{PNG: png}, nil
}
