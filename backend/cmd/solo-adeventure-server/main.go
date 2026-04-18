package main

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/adapters/auth"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/depth"
	httpadapter "github.com/ridopark/solo-adeventure/backend/internal/adapters/http"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/image"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/llm"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/notifier"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/store"
	"github.com/ridopark/solo-adeventure/backend/internal/adapters/tts"
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

	db, err := store.NewSQLite(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Str("path", cfg.DBPath).Msg("open sqlite")
	}
	defer db.Close()

	storyProvider := buildStoryProvider(cfg, llmClient, log)
	imageProvider := buildImageProvider(cfg, imgClient, log)
	notif := buildNotifier(cfg, log)

	oauthProvider := buildOAuth(cfg, log)

	svc := app.NewService(db, storyProvider, imageProvider, notif, log).
		WithAuth(db.Users(), db.Sessions(), oauthProvider)

	if cfg.TTSEnabled {
		svc = svc.WithTTS(tts.NewEdge(cfg.TTSURL, cfg.TTSVoice, log), cfg.AudioDir, cfg.AudioURLBase)
		log.Info().Str("tts_url", cfg.TTSURL).Str("voice", cfg.TTSVoice).Msg("tts sidecar wired")
	} else {
		log.Info().Msg("TTS_ENABLED=false; narration disabled")
	}

	if cfg.DepthEnabled {
		svc = svc.WithDepth(depth.NewLocal(cfg.DepthURL, log), cfg.DepthDir, cfg.DepthURLBase, cfg.PublicBaseURL)
		log.Info().Str("depth_url", cfg.DepthURL).Msg("depth sidecar wired")
	} else {
		log.Info().Msg("DEPTH_ENABLED=false; parallax disabled")
	}

	handler := httpadapter.NewRouter(svc, log, httpadapter.RouterConfig{
		CORSOrigin:   cfg.CORSAllowOrigin,
		FrontendURL:  cfg.FrontendURL,
		CookieDomain: cfg.CookieDomain,
		Secure:       cfg.CookieSecure,
		AudioDir:     cfg.AudioDir,
		DepthDir:     cfg.DepthDir,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Info().Str("port", cfg.Port).Str("db", cfg.DBPath).Msg("solo-adeventure-server listening")
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

func buildOAuth(cfg config.Config, log zerolog.Logger) ports.OAuthProvider {
	if cfg.OAuthProvider == "stub" || cfg.GoogleOAuthClientID == "" {
		log.Warn().Msg("using stub oauth provider -- set GOOGLE_OAUTH_CLIENT_ID to enable real Google sign-in")
		return auth.NewStub(cfg.FrontendURL)
	}
	return auth.NewGoogle(cfg.GoogleOAuthClientID, cfg.GoogleOAuthSecret, cfg.GoogleRedirectURI, nil)
}
