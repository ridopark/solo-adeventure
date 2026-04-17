package image

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

// Fallback tries Primary first; on a transient error (rate limit, 5xx, timeout)
// it invokes Fallback. A non-transient error from Primary is returned without
// invoking the fallback.
type Fallback struct {
	Primary  ports.ImageProvider
	Fallback ports.ImageProvider
	Log      zerolog.Logger
}

func NewFallback(primary, fallback ports.ImageProvider, log zerolog.Logger) *Fallback {
	return &Fallback{
		Primary:  primary,
		Fallback: fallback,
		Log:      log.With().Str("component", "image.fallback").Logger(),
	}
}

func (f *Fallback) Generate(ctx context.Context, req ports.ImageRequest) (ports.ImageResult, error) {
	res, err := f.Primary.Generate(ctx, req)
	if err == nil {
		return res, nil
	}
	if !IsTransient(err) {
		return ports.ImageResult{}, fmt.Errorf("image.fallback: primary permanent: %w", err)
	}
	f.Log.Warn().Err(err).Msg("primary image provider transient error; falling back")
	if f.Fallback == nil {
		return ports.ImageResult{}, fmt.Errorf("image.fallback: no fallback configured: %w", err)
	}
	res, err = f.Fallback.Generate(ctx, req)
	if err != nil {
		return ports.ImageResult{}, fmt.Errorf("image.fallback: fallback failed: %w", err)
	}
	return res, nil
}
