package image

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

type stubProvider struct {
	name string
	res  ports.ImageResult
	err  error
	hits int
}

func (s *stubProvider) Generate(ctx context.Context, req ports.ImageRequest) (ports.ImageResult, error) {
	s.hits++
	if s.err != nil {
		return ports.ImageResult{}, s.err
	}
	return s.res, nil
}

func TestFallback_Generate(t *testing.T) {
	tests := []struct {
		name         string
		primaryRes   ports.ImageResult
		primaryErr   error
		fallbackRes  ports.ImageResult
		fallbackErr  error
		wantProvider string
		wantURL      string
		wantErr      bool
		wantPrimary  int
		wantFallback int
	}{
		{
			name:         "primary ok -- fallback not invoked",
			primaryRes:   ports.ImageResult{URL: "https://prim/1.png", Provider: "together"},
			wantProvider: "together",
			wantURL:      "https://prim/1.png",
			wantPrimary:  1,
			wantFallback: 0,
		},
		{
			name:         "primary transient 429 -- fallback returns",
			primaryErr:   fmt.Errorf("image.together: status 429: %w", ErrTransient),
			fallbackRes:  ports.ImageResult{URL: "https://fb/2.png", Provider: "fal"},
			wantProvider: "fal",
			wantURL:      "https://fb/2.png",
			wantPrimary:  1,
			wantFallback: 1,
		},
		{
			name:         "primary transient 503 -- fallback returns",
			primaryErr:   fmt.Errorf("image.together: status 503: %w", ErrTransient),
			fallbackRes:  ports.ImageResult{URL: "https://fb/3.png", Provider: "fal"},
			wantProvider: "fal",
			wantURL:      "https://fb/3.png",
			wantPrimary:  1,
			wantFallback: 1,
		},
		{
			name:         "primary permanent 400 -- fallback NOT invoked",
			primaryErr:   errors.New("image.together: status 400: permanent"),
			wantErr:      true,
			wantPrimary:  1,
			wantFallback: 0,
		},
		{
			name:         "primary transient, fallback also fails -- error surfaced",
			primaryErr:   fmt.Errorf("image.together: status 429: %w", ErrTransient),
			fallbackErr:  errors.New("image.fal: blew up"),
			wantErr:      true,
			wantPrimary:  1,
			wantFallback: 1,
		},
		{
			name:         "primary context canceled -- treated transient, fallback invoked",
			primaryErr:   context.Canceled,
			fallbackRes:  ports.ImageResult{URL: "https://fb/5.png", Provider: "fal"},
			wantErr:      true,
			wantPrimary:  1,
			wantFallback: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			primary := &stubProvider{name: "primary", res: tc.primaryRes, err: tc.primaryErr}
			fallback := &stubProvider{name: "fallback", res: tc.fallbackRes, err: tc.fallbackErr}
			fb := NewFallback(primary, fallback, zerolog.Nop())

			res, err := fb.Generate(context.Background(), ports.ImageRequest{Prompt: "x"})
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantURL, res.URL)
				assert.Equal(t, tc.wantProvider, res.Provider)
			}
			assert.Equal(t, tc.wantPrimary, primary.hits, "primary hit count")
			assert.Equal(t, tc.wantFallback, fallback.hits, "fallback hit count")
		})
	}

	t.Run("nil fallback returns wrapped error on primary transient", func(t *testing.T) {
		primary := &stubProvider{err: fmt.Errorf("image.together: status 503: %w", ErrTransient)}
		fb := NewFallback(primary, nil, zerolog.Nop())
		_, err := fb.Generate(context.Background(), ports.ImageRequest{})
		assert.Error(t, err)
		assert.Equal(t, 1, primary.hits)
	})
}
