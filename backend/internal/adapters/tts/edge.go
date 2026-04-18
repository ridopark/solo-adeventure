package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

// voicesByLanguage maps BCP-47 tags to edge-tts (Azure Neural) voice names.
// Add a locale-less key for common languages so "ko" also resolves.
var voicesByLanguage = map[string]string{
	"en-US": "en-US-AndrewNeural",
	"en-GB": "en-GB-RyanNeural",
	"en":    "en-US-AndrewNeural",
	"ko-KR": "ko-KR-InJoonNeural",
	"ko":    "ko-KR-InJoonNeural",
	"ja-JP": "ja-JP-KeitaNeural",
	"ja":    "ja-JP-KeitaNeural",
	"zh-CN": "zh-CN-YunxiNeural",
	"zh-TW": "zh-TW-YunJheNeural",
	"zh":    "zh-CN-YunxiNeural",
	"es-ES": "es-ES-AlvaroNeural",
	"es-MX": "es-MX-JorgeNeural",
	"es":    "es-ES-AlvaroNeural",
	"fr-FR": "fr-FR-HenriNeural",
	"fr":    "fr-FR-HenriNeural",
	"de-DE": "de-DE-ConradNeural",
	"de":    "de-DE-ConradNeural",
	"it-IT": "it-IT-DiegoNeural",
	"it":    "it-IT-DiegoNeural",
	"pt-BR": "pt-BR-AntonioNeural",
	"pt-PT": "pt-PT-DuarteNeural",
	"pt":    "pt-BR-AntonioNeural",
	"nl-NL": "nl-NL-MaartenNeural",
	"nl":    "nl-NL-MaartenNeural",
	"ru-RU": "ru-RU-DmitryNeural",
	"ru":    "ru-RU-DmitryNeural",
	"pl-PL": "pl-PL-MarekNeural",
	"pl":    "pl-PL-MarekNeural",
	"tr-TR": "tr-TR-AhmetNeural",
	"tr":    "tr-TR-AhmetNeural",
	"ar-SA": "ar-SA-HamedNeural",
	"ar":    "ar-SA-HamedNeural",
	"hi-IN": "hi-IN-MadhurNeural",
	"hi":    "hi-IN-MadhurNeural",
	"vi-VN": "vi-VN-NamMinhNeural",
	"vi":    "vi-VN-NamMinhNeural",
	"th-TH": "th-TH-NiwatNeural",
	"th":    "th-TH-NiwatNeural",
	"id-ID": "id-ID-ArdiNeural",
	"id":    "id-ID-ArdiNeural",
}

func voiceForLanguage(lang string) string {
	if lang == "" {
		return ""
	}
	tag := strings.TrimSpace(lang)
	if v, ok := voicesByLanguage[tag]; ok {
		return v
	}
	// Fall back to the primary subtag: "ko-KR" -> "ko"
	if dash := strings.Index(tag, "-"); dash > 0 {
		if v, ok := voicesByLanguage[tag[:dash]]; ok {
			return v
		}
	}
	return ""
}

type Edge struct {
	baseURL string
	voice   string
	client  *http.Client
	log     zerolog.Logger
}

func NewEdge(baseURL, voice string, log zerolog.Logger) *Edge {
	return &Edge{
		baseURL: baseURL,
		voice:   voice,
		client:  &http.Client{Timeout: 120 * time.Second},
		log:     log.With().Str("component", "tts.edge").Logger(),
	}
}

type edgeReqBody struct {
	Text  string `json:"text"`
	Voice string `json:"voice,omitempty"`
}

func (e *Edge) Synthesize(ctx context.Context, req ports.TTSRequest) (ports.TTSResult, error) {
	voice := req.Voice
	if voice == "" {
		voice = voiceForLanguage(req.Language)
	}
	if voice == "" {
		voice = e.voice
	}
	body, err := json.Marshal(edgeReqBody{Text: req.Text, Voice: voice})
	if err != nil {
		return ports.TTSResult{}, fmt.Errorf("tts: marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/tts", bytes.NewReader(body))
	if err != nil {
		return ports.TTSResult{}, fmt.Errorf("tts: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(httpReq)
	if err != nil {
		return ports.TTSResult{}, fmt.Errorf("tts: sidecar unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return ports.TTSResult{}, fmt.Errorf("tts: sidecar status %d: %s", resp.StatusCode, string(snippet))
	}
	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return ports.TTSResult{}, fmt.Errorf("tts: read audio: %w", err)
	}
	if len(audio) == 0 {
		return ports.TTSResult{}, errors.New("tts: empty audio payload")
	}
	return ports.TTSResult{Audio: audio, Format: "audio/mpeg"}, nil
}
