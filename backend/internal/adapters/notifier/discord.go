package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

type Discord struct {
	WebhookURL string
	Client     *http.Client
	Log        zerolog.Logger
	Username   string
}

func NewDiscord(webhookURL string, log zerolog.Logger) *Discord {
	return &Discord{
		WebhookURL: webhookURL,
		Client:     &http.Client{Timeout: 3 * time.Second},
		Log:        log.With().Str("component", "notifier.discord").Logger(),
		Username:   "solo-adeventure",
	}
}

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type discordEmbedImage struct {
	URL string `json:"url"`
}

type discordEmbedAuthor struct {
	Name    string `json:"name"`
	IconURL string `json:"icon_url,omitempty"`
}

type discordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []discordEmbedField `json:"fields,omitempty"`
	Image       *discordEmbedImage  `json:"image,omitempty"`
	Author      *discordEmbedAuthor `json:"author,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type discordPayload struct {
	Username string         `json:"username,omitempty"`
	Embeds   []discordEmbed `json:"embeds"`
}

func colorFor(kind ports.NotifyKind) int {
	switch kind {
	case ports.NotifyTopicSubmitted:
		return 0x2563eb
	case ports.NotifyPageGenerated:
		return 0x57534e
	case ports.NotifyImageGenerated:
		return 0x059669
	case ports.NotifyChoiceMade:
		return 0xd97706
	case ports.NotifyVisitStarted:
		return 0x8b5cf6
	}
	return 0x737373
}

// Notify fires an embed to Discord. The send is fully non-blocking -- the
// caller's goroutine returns immediately; webhook failures are logged as
// warnings and never propagate.
func (d *Discord) Notify(ev ports.NotifyEvent) {
	if d.WebhookURL == "" {
		return
	}
	go d.sendEmbed(ev)
}

func (d *Discord) sendEmbed(ev ports.NotifyEvent) {
	defer func() {
		if r := recover(); r != nil {
			d.Log.Warn().Interface("panic", r).Msg("discord notify panic recovered")
		}
	}()

	embed := discordEmbed{
		Title:       ev.Title,
		Description: ev.Message,
		Color:       colorFor(ev.Kind),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	if ev.UserName != "" {
		embed.Author = &discordEmbedAuthor{Name: ev.UserName, IconURL: ev.UserAvatarURL}
	}
	for _, f := range ev.Fields {
		embed.Fields = append(embed.Fields, discordEmbedField{
			Name:   f.Name,
			Value:  f.Value,
			Inline: len(f.Value) < 40,
		})
	}
	if ev.StoryID != "" {
		embed.Fields = append(embed.Fields, discordEmbedField{
			Name:   "story",
			Value:  fmt.Sprintf("`%s`", ev.StoryID),
			Inline: true,
		})
	}
	if ev.ImageURL != "" {
		embed.Image = &discordEmbedImage{URL: ev.ImageURL}
	}

	body, err := json.Marshal(discordPayload{Username: d.Username, Embeds: []discordEmbed{embed}})
	if err != nil {
		d.Log.Warn().Err(err).Msg("discord marshal failed")
		return
	}
	req, err := http.NewRequest(http.MethodPost, d.WebhookURL, bytes.NewReader(body))
	if err != nil {
		d.Log.Warn().Err(err).Msg("discord build request failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.Client.Do(req)
	if err != nil {
		d.Log.Warn().Err(err).Msg("discord webhook failed")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		d.Log.Warn().Int("status", resp.StatusCode).Msg("discord non-2xx")
	}
}
