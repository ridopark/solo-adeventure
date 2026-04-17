package image

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

func TestTogether_Generate(t *testing.T) {
	t.Parallel()

	type captured struct {
		mu         sync.Mutex
		authHeader string
		body       map[string]any
	}

	t.Run("happy path returns url and captures request", func(t *testing.T) {
		t.Parallel()
		cap := &captured{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cap.mu.Lock()
			defer cap.mu.Unlock()
			cap.authHeader = r.Header.Get("Authorization")
			raw, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(raw, &cap.body))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"url":"https://cdn.example/image.png"}]}`))
		}))
		defer srv.Close()

		tg := NewTogether("sk-test", srv.Client(), zerolog.Nop())
		tg.Endpoint = srv.URL

		res, err := tg.Generate(context.Background(), ports.ImageRequest{
			Prompt:      "a brave knight at sunset",
			StylePrefix: domain.StylePrefix("watercolor storybook"),
		})
		require.NoError(t, err)
		assert.Equal(t, "https://cdn.example/image.png", res.URL)
		assert.Equal(t, "together", res.Provider)

		cap.mu.Lock()
		defer cap.mu.Unlock()
		assert.Equal(t, "Bearer sk-test", cap.authHeader)
		assert.Equal(t, "black-forest-labs/FLUX.1-schnell", cap.body["model"])
		assert.Equal(t, "url", cap.body["response_format"])
		assert.EqualValues(t, 1024, cap.body["width"])
		assert.EqualValues(t, 1024, cap.body["height"])
		assert.EqualValues(t, 4, cap.body["steps"])
		assert.EqualValues(t, 1, cap.body["n"])

		prompt, ok := cap.body["prompt"].(string)
		require.True(t, ok, "prompt must be string")
		assert.Contains(t, prompt, "watercolor storybook")
		assert.Contains(t, prompt, "a brave knight at sunset")
		assert.Contains(t, prompt, "no text, no letters, no words.")
	})

	t.Run("empty style prefix omitted from prompt", func(t *testing.T) {
		t.Parallel()
		var gotPrompt string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			if p, ok := body["prompt"].(string); ok {
				gotPrompt = p
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"url":"https://x/y.png"}]}`))
		}))
		defer srv.Close()

		tg := NewTogether("k", srv.Client(), zerolog.Nop())
		tg.Endpoint = srv.URL

		_, err := tg.Generate(context.Background(), ports.ImageRequest{
			Prompt:      "a dragon",
			StylePrefix: "",
		})
		require.NoError(t, err)
		assert.False(t, strings.HasPrefix(gotPrompt, ". "), "no leading separator when style empty")
		assert.Contains(t, gotPrompt, "a dragon")
		assert.Contains(t, gotPrompt, "no text, no letters, no words.")
	})

	statusCases := []struct {
		name      string
		status    int
		transient bool
	}{
		{name: "429 is transient", status: http.StatusTooManyRequests, transient: true},
		{name: "500 is transient", status: http.StatusInternalServerError, transient: true},
		{name: "503 is transient", status: http.StatusServiceUnavailable, transient: true},
		{name: "400 is permanent", status: http.StatusBadRequest, transient: false},
		{name: "401 is permanent", status: http.StatusUnauthorized, transient: false},
		{name: "404 is permanent", status: http.StatusNotFound, transient: false},
	}
	for _, tc := range statusCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":"boom"}`))
			}))
			defer srv.Close()

			tg := NewTogether("k", srv.Client(), zerolog.Nop())
			tg.Endpoint = srv.URL

			_, err := tg.Generate(context.Background(), ports.ImageRequest{Prompt: "p"})
			require.Error(t, err)
			if tc.transient {
				assert.True(t, errors.Is(err, ErrTransient), "expected transient, got %v", err)
				assert.True(t, IsTransient(err))
			} else {
				assert.False(t, errors.Is(err, ErrTransient), "expected permanent, got %v", err)
			}
		})
	}

	t.Run("200 with empty data array returns permanent error", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		}))
		defer srv.Close()

		tg := NewTogether("k", srv.Client(), zerolog.Nop())
		tg.Endpoint = srv.URL

		_, err := tg.Generate(context.Background(), ports.ImageRequest{Prompt: "p"})
		require.Error(t, err)
		assert.False(t, errors.Is(err, ErrTransient))
		assert.Contains(t, err.Error(), "image.together")
	})

	t.Run("200 with empty url in data returns permanent error", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"url":""}]}`))
		}))
		defer srv.Close()

		tg := NewTogether("k", srv.Client(), zerolog.Nop())
		tg.Endpoint = srv.URL

		_, err := tg.Generate(context.Background(), ports.ImageRequest{Prompt: "p"})
		require.Error(t, err)
		assert.False(t, errors.Is(err, ErrTransient))
	})

	t.Run("context cancellation returns error", func(t *testing.T) {
		t.Parallel()
		release := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-release:
			case <-r.Context().Done():
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		defer close(release)

		tg := NewTogether("k", srv.Client(), zerolog.Nop())
		tg.Endpoint = srv.URL

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		_, err := tg.Generate(ctx, ports.ImageRequest{Prompt: "slow"})
		require.Error(t, err)
		assert.True(t,
			errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded),
			"expected context error, got %v", err,
		)
	})
}
