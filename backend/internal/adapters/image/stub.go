package image

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

// Stub is a deterministic ImageProvider for local dev and E2E without a real image API.
// It returns a stable placeholder URL that varies by prompt so each page gets a different image.
type Stub struct {
	BaseURL string
}

func NewStub() *Stub {
	return &Stub{BaseURL: "https://picsum.photos/seed"}
}

func (s *Stub) Generate(_ context.Context, req ports.ImageRequest) (ports.ImageResult, error) {
	h := sha1.Sum([]byte(string(req.StylePrefix) + "|" + req.Prompt))
	seed := hex.EncodeToString(h[:6])
	url := fmt.Sprintf("%s/%s/1024/768", s.BaseURL, seed)
	return ports.ImageResult{URL: url, Provider: "stub"}, nil
}
