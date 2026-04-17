package image

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

func TestFal_Generate(t *testing.T) {
	t.Parallel()

	const apiKey = "test-key-abc"

	t.Run("happy path returns URL and provider", func(t *testing.T) {
		t.Parallel()

		var gotAuth string
		var gotContentType string
		var gotBody map[string]any

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			gotContentType = r.Header.Get("Content-Type")
			assert.Equal(t, http.MethodPost, r.Method)
			raw, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(raw, &gotBody))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"images":[{"url":"https://cdn.fal.example/out.png"}]}`))
		}))
		defer srv.Close()

		f := NewFal(apiKey, srv.Client(), zerolog.Nop())
		f.Endpoint = srv.URL

		res, err := f.Generate(context.Background(), ports.ImageRequest{
			Prompt:      "a brave hero in a forest",
			StylePrefix: domain.StylePrefix("cinematic watercolor"),
		})
		require.NoError(t, err)
		assert.Equal(t, "https://cdn.fal.example/out.png", res.URL)
		assert.Equal(t, "fal", res.Provider)

		assert.Equal(t, "Key "+apiKey, gotAuth)
		assert.NotContains(t, gotAuth, "Bearer")
		assert.Equal(t, "application/json", gotContentType)

		prompt, _ := gotBody["prompt"].(string)
		assert.Contains(t, prompt, "cinematic watercolor")
		assert.Contains(t, prompt, "a brave hero in a forest")
		assert.Contains(t, prompt, "no text")
		assert.True(t, strings.Contains(prompt, ". "), "prompt parts should be joined by '. '")

		assert.Equal(t, "square_hd", gotBody["image_size"])
		assert.EqualValues(t, 4, gotBody["num_inference_steps"])
		assert.EqualValues(t, 1, gotBody["num_images"])
		assert.Equal(t, true, gotBody["enable_safety_checker"])
	})

	t.Run("empty style prefix omits style segment", func(t *testing.T) {
		t.Parallel()

		var gotBody map[string]any
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(raw, &gotBody))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"images":[{"url":"https://cdn.fal.example/x.png"}]}`))
		}))
		defer srv.Close()

		f := NewFal(apiKey, srv.Client(), zerolog.Nop())
		f.Endpoint = srv.URL

		_, err := f.Generate(context.Background(), ports.ImageRequest{
			Prompt:      "a lone tree",
			StylePrefix: "",
		})
		require.NoError(t, err)

		prompt, _ := gotBody["prompt"].(string)
		assert.Contains(t, prompt, "a lone tree")
		assert.Contains(t, prompt, "no text")
		assert.False(t, strings.HasPrefix(prompt, ". "), "should not leak empty style as leading separator")
	})

	t.Run("429 is transient", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limited"}`))
		}))
		defer srv.Close()

		f := NewFal(apiKey, srv.Client(), zerolog.Nop())
		f.Endpoint = srv.URL

		_, err := f.Generate(context.Background(), ports.ImageRequest{Prompt: "x"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrTransient), "429 should map to ErrTransient, got: %v", err)
		assert.True(t, IsTransient(err))
	})

	t.Run("503 is transient", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		f := NewFal(apiKey, srv.Client(), zerolog.Nop())
		f.Endpoint = srv.URL

		_, err := f.Generate(context.Background(), ports.ImageRequest{Prompt: "x"})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrTransient))
	})

	t.Run("401 is permanent", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"bad key"}`))
		}))
		defer srv.Close()

		f := NewFal(apiKey, srv.Client(), zerolog.Nop())
		f.Endpoint = srv.URL

		_, err := f.Generate(context.Background(), ports.ImageRequest{Prompt: "x"})
		require.Error(t, err)
		assert.False(t, errors.Is(err, ErrTransient), "401 must not be transient: %v", err)
		assert.False(t, IsTransient(err))
	})

	t.Run("200 with empty images returns permanent error", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"images":[]}`))
		}))
		defer srv.Close()

		f := NewFal(apiKey, srv.Client(), zerolog.Nop())
		f.Endpoint = srv.URL

		_, err := f.Generate(context.Background(), ports.ImageRequest{Prompt: "x"})
		require.Error(t, err)
		assert.False(t, errors.Is(err, ErrTransient))
		assert.False(t, IsTransient(err))
		assert.Contains(t, err.Error(), "image.fal")
	})

	t.Run("context cancellation returns error", func(t *testing.T) {
		t.Parallel()

		block := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-block:
			case <-r.Context().Done():
			}
		}))
		defer srv.Close()
		defer close(block)

		f := NewFal(apiKey, srv.Client(), zerolog.Nop())
		f.Endpoint = srv.URL

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := f.Generate(ctx, ports.ImageRequest{Prompt: "x"})
		require.Error(t, err)
	})
}
