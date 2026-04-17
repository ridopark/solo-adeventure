package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	LogLevel          string
	CORSAllowOrigin   string
	AnthropicAPIKey   string
	AnthropicModel    string
	TogetherAPIKey    string
	FalKey            string
	StoryProvider     string
	ImagePrimary      string
	ImageFallback     string
	LLMTimeout        time.Duration
	ImageTimeout      time.Duration
	DiscordWebhookURL   string
	DBPath              string
	FrontendURL         string
	CookieDomain        string
	CookieSecure        bool
	GoogleOAuthClientID string
	GoogleOAuthSecret   string
	GoogleRedirectURI   string
	OAuthProvider       string
}

func Load() Config {
	return Config{
		Port:            getenv("PORT", "8084"),
		LogLevel:        getenv("LOG_LEVEL", "info"),
		CORSAllowOrigin: getenv("CORS_ALLOW_ORIGIN", "http://localhost:3004"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:  getenv("ANTHROPIC_MODEL", "claude-haiku-4-5"),
		TogetherAPIKey:  os.Getenv("TOGETHER_API_KEY"),
		FalKey:          os.Getenv("FAL_KEY"),
		StoryProvider:   getenv("STORY_PROVIDER", "anthropic"),
		ImagePrimary:    getenv("IMAGE_PRIMARY", "together"),
		ImageFallback:   getenv("IMAGE_FALLBACK", "fal"),
		LLMTimeout:         getDuration("LLM_TIMEOUT", 45*time.Second),
		ImageTimeout:       getDuration("IMAGE_TIMEOUT", 30*time.Second),
		DiscordWebhookURL:  os.Getenv("DISCORD_WEBHOOK_URL"),
		DBPath:              getenv("DB_PATH", "/opt/solo-adeventure/solo-adeventure.db"),
		FrontendURL:         getenv("FRONTEND_URL", "http://localhost:3004"),
		CookieDomain:        os.Getenv("COOKIE_DOMAIN"),
		CookieSecure:        getenv("COOKIE_SECURE", "true") == "true",
		GoogleOAuthClientID: os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
		GoogleOAuthSecret:   os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
		GoogleRedirectURI:   getenv("GOOGLE_REDIRECT_URI", "http://localhost:8084/auth/google/callback"),
		OAuthProvider:       getenv("OAUTH_PROVIDER", "google"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	return def
}
