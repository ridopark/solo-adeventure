package main

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"

	httpadapter "github.com/ridopark/solo-adeventure/backend/internal/adapters/http"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/image"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/llm"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/notifier"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/store"
	"github.com/ridopark/solo-adeventure/backend/internal/app"
	"github.com/ridopark/solo-adeventure/backend/internal/config"
	"github.com/ridopark/solo-adeventure/backend/internal/logger"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	imgClient := &http.Client{Timeout: cfg.ImageTimeout}
	llmClient := &http.Client{Timeout: cfg.LLMTimeout}

	storyStore := store.NewMemory()
	storyProvider := buildStoryProvider(cfg, llmClient, log)
	imageProvider := buildImageProvider(cfg, imgClient, log)
	notif := buildNotifier(cfg, log)

	svc := app.NewService(storyStore, storyProvider, imageProvider, notif, log)

	handler := httpadapter.NewRouter(svc, log, cfg.CORSAllowOrigin)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Info().Str("port", cfg.Port).Msg("solo-adeventure-server listening")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Msg("server terminated")
	}
}

func buildImageProvider(cfg config.Config, client *http.Client, log zerolog.Logger) ports.ImageProvider {
	primary := buildOne(cfg.ImagePrimary, cfg, client, log)
	if cfg.ImageFallback == "" {
		return primary
	}
	fallback := buildOne(cfg.ImageFallback, cfg, client, log)
	return image.NewFallback(primary, fallback, log)
}

func buildOne(name string, cfg config.Config, client *http.Client, log zerolog.Logger) ports.ImageProvider {
	switch name {
	case "together":
		return image.NewTogether(cfg.TogetherAPIKey, client, log)
	case "fal":
		return image.NewFal(cfg.FalKey, client, log)
	case "stub":
		return image.NewStub()
	default:
		log.Fatal().Str("provider", name).Msg("unknown image provider")
		return nil
	}
}

func buildNotifier(cfg config.Config, log zerolog.Logger) ports.Notifier {
	if cfg.DiscordWebhookURL == "" {
		log.Info().Msg("DISCORD_WEBHOOK_URL unset; notifications disabled")
		return notifier.NewNoop()
	}
	return notifier.NewDiscord(cfg.DiscordWebhookURL, log)
}

func buildStoryProvider(cfg config.Config, client *http.Client, log zerolog.Logger) ports.StoryProvider {
	if cfg.StoryProvider == "stub" {
		log.Warn().Msg("using stub story provider -- set STORY_PROVIDER=anthropic for real LLM")
		return llm.NewStub()
	}
	return llm.NewAnthropic(cfg.AnthropicAPIKey, cfg.AnthropicModel, client, log)
}
