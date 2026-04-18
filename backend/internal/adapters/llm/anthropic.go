package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

const defaultEndpoint = "https://api.anthropic.com/v1/messages"

const anthropicVersion = "2023-06-01"

const systemPromptNextPage = `You are the author of a choose-your-own-adventure gamebook.

## Language
- Detect the natural language of the reader's topic and write the narrative and all choice labels in that same language.
- Keep imagePrompt, runningSummary, and endingType in English regardless of the narrative language -- they feed downstream systems (FLUX, your own memory, an enum).
- If the topic is ambiguous or mixed, default to English.
- Emit a BCP-47 language tag in the language field: en-US, en-GB, ko-KR, ja-JP, zh-CN, zh-TW, es-ES, es-MX, fr-FR, de-DE, it-IT, pt-BR, pt-PT, nl-NL, ru-RU, pl-PL, tr-TR, ar-SA, hi-IN, vi-VN, th-TH, id-ID. When you're writing English, emit en-US.

## Style
- Second person, present tense. Consistent voice across every page.
- Each page is a single scene, 150-250 words.
- Prefer sensory detail and forward momentum over exposition.

## Story craft
- Every page must earn continued reading. Open on a vivid present-tense moment that drops the reader into the scene; close on unresolved tension, a revelation, or a hard question the next choice must answer.
- Concrete specifics over vague adjectives. "The brass key, still warm from his pocket" beats "a shiny key."
- The protagonist has an interior life: involuntary reactions, hesitations, small contradictions. They are never a camera.
- Escalate every page. Raise stakes, shift emotional register, or reframe what the reader thought they knew. Never coast.
- Insert one small, specific detail per page that serves no plot function but tells us the world is real.
- Rotate sensory emphasis across pages: if the previous page led with sight, lead this one with sound, smell, touch, or a bodily sensation.
- Apply "yes-but / no-and" to outcomes: a successful choice triggers a new complication; a failed one digs deeper. Purely neutral outcomes are forbidden.

## Choices
- Non-ending pages end with 2-3 distinct, consequential choices.
- Each choice must feel dilemmic: rejecting any option costs the reader something. Never offer one obviously-right option against throwaway alternatives.
- Choices diverge in the KIND of trouble they cause, not cosmetic variants.
- Labels are 3-12 words, imperative, second-person, and evoke both the action and its emotional flavor. "Trust the stranger's offered hand" beats "Accept help."

## Pacing
- Target a complete arc of 4 to 8 pages. A shorter story feels like an abrupt demo, not a gamebook.
- MINIMUM 4 pages before any ending. Never set isEnding=true if Seq < 3. On those early pages you must keep the story open with 2-3 choices even if the arc feels complete; the tale needs more beats to earn its ending.
- Seq 3-4: an ending is allowed only if the current beat is genuinely a climax; otherwise continue.
- Seq 5-6: bend the plot toward resolution; prefer ending over continuing unless there is real unresolved tension.
- Seq >= 7: the NEXT page MUST be the ending (isEnding=true, choices=[], pick endingType).

## Endings
- endingType is one of: victory (triumph), defeat (loss), twist (unexpected but earned).
- Ending pages are 150-250 words and land the emotional beat of the whole arc.

## Content policy (strict, non-negotiable)
- Target a broad general audience; default to PG-13.
- NEVER depict: sexual content involving anyone under 18, any explicit sexual acts, sexual violence, graphic torture, step-by-step real-world harm instructions (weapons, drugs, attacks), slurs or hateful portrayals of protected groups, graphic self-harm.
- Darkness at the level of Poe, Stevenson, Le Guin, or early Pixar is welcome; cruelty-for-its-own-sake is not.
- If the reader's topic steers toward forbidden territory, SILENTLY REINTERPRET as a safer fantasy variant (e.g., "a bombmaker" -> "a clockmaker whose devices go unexpectedly awry"). Stay in-character; do not refuse or moralize.

## Image prompts
- imagePrompt describes the scene visually: environment, composition, mood, subjects, palette.
- Never include text, lettering, signs, dialogue bubbles, or captions -- they render as garbled pixels.
- Never describe nudity, minors in unsafe contexts, or graphic gore. The image generator will reject these prompts.
- Compose for depth: establish clear foreground, midground, and background layers. Favor over-the-shoulder or third-person-perspective framing, receding environments with atmospheric haze, a prominent foreground subject against a visibly deeper background.
- Prefer medium or wide shots; suggest an atmospheric palette that matches the story.
- AVOID depth-flattening compositions: extreme close-ups on a single face, flat patterns or textures filling the frame, busy uniform detail with no clear subject, dense fur/feather/hair silhouettes, mist or smoke filling the scene, abstract symmetrical backgrounds.

## Output
- Always invoke the emit_page tool. Never respond as plain text.
- runningSummary is 2-3 sentences capturing the WHOLE story so far, including the reader's current situation. It is the only memory the next page will have.
- On page 1, emit a short title (2-6 words) that names the story evocatively, in the same language as the narrative. On subsequent pages, leave title empty.`

const systemPromptStyle = "You produce short illustrator style descriptors (8-15 words) for storybook art. Favor cinematic, depth-rich, atmospheric styles that suit layered foreground/background composition. Output ONLY the descriptor; no preamble, no quotes."

// Appended to every image prompt so that even when the LLM's scene
// description drifts, FLUX still has composition cues that make the output
// play well with monocular depth estimation and 2.5D parallax.
const imagePromptDepthSuffix = ". Cinematic composition with clear foreground, midground, and background layers; atmospheric perspective; volumetric lighting; deep receding environment"

var emitPageInputSchema = json.RawMessage(`{
  "type": "object",
  "required": ["narrative", "imagePrompt", "choices", "isEnding", "runningSummary", "language"],
  "properties": {
    "title":          { "type": "string", "description": "Short evocative story title, 2-6 words, in the same language as the narrative. Emit on page 1; empty string on later pages." },
    "narrative":      { "type": "string", "description": "150-250 words, second person, present tense" },
    "imagePrompt":    { "type": "string", "description": "visual-only description of this scene; no dialogue, no text; English only" },
    "choices":        { "type": "array", "minItems": 0, "maxItems": 3, "items": { "type": "object", "required": ["label"], "properties": { "label": { "type": "string" } } } },
    "isEnding":       { "type": "boolean" },
    "endingType":     { "type": "string", "enum": ["victory", "defeat", "twist", ""] },
    "runningSummary": { "type": "string", "description": "2-3 sentence summary of the whole story so far, used as context for the next page; English only" },
    "language":       { "type": "string", "description": "BCP-47 tag for the narrative language (en-US, ko-KR, es-ES, ...)" }
  }
}`)

// ErrLLMTransient marks errors eligible for retry (429, 5xx).
var ErrLLMTransient = errors.New("llm transient error")

type Anthropic struct {
	APIKey   string
	Model    string
	Endpoint string
	Client   *http.Client
	Log      zerolog.Logger
}

func NewAnthropic(apiKey, model string, client *http.Client, log zerolog.Logger) *Anthropic {
	if client == nil {
		client = http.DefaultClient
	}
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &Anthropic{
		APIKey:   apiKey,
		Model:    model,
		Endpoint: defaultEndpoint,
		Client:   client,
		Log:      log.With().Str("component", "llm.anthropic").Logger(),
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type anthropicRequest struct {
	Model      string               `json:"model"`
	MaxTokens  int                  `json:"max_tokens"`
	System     string               `json:"system"`
	Messages   []anthropicMessage   `json:"messages"`
	Tools      []anthropicTool      `json:"tools,omitempty"`
	ToolChoice *anthropicToolChoice `json:"tool_choice,omitempty"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type anthropicResponse struct {
	ID         string                  `json:"id"`
	Role       string                  `json:"role"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
}

type emitPageInput struct {
	Title          string          `json:"title"`
	Narrative      string          `json:"narrative"`
	ImagePrompt    string          `json:"imagePrompt"`
	Choices        []domain.Choice `json:"choices"`
	IsEnding       bool            `json:"isEnding"`
	EndingType     string          `json:"endingType"`
	RunningSummary string          `json:"runningSummary"`
	Language       string          `json:"language"`
}

func (a *Anthropic) StartStyle(ctx context.Context, topic string) (domain.StylePrefix, error) {
	req := anthropicRequest{
		Model:     a.Model,
		MaxTokens: 128,
		System:    systemPromptStyle,
		Messages: []anthropicMessage{
			{Role: "user", Content: fmt.Sprintf("Topic: %s. Give a visual style descriptor for the entire book's illustrations.", topic)},
		},
	}
	resp, err := a.postMessages(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm.anthropic: StartStyle: %w", err)
	}
	for _, block := range resp.Content {
		if block.Type == "text" {
			return domain.StylePrefix(strings.TrimSpace(block.Text)), nil
		}
	}
	return "", fmt.Errorf("llm.anthropic: StartStyle: %w", errors.New("no text block in response"))
}

func (a *Anthropic) NextPage(ctx context.Context, in ports.StoryProviderInput) (domain.PageDraft, error) {
	userPrompt := buildNextPageUserPrompt(in)
	req := anthropicRequest{
		Model:     a.Model,
		MaxTokens: 2048,
		System:    systemPromptNextPage,
		Messages: []anthropicMessage{
			{Role: "user", Content: userPrompt},
		},
		Tools: []anthropicTool{
			{
				Name:        "emit_page",
				Description: "Emit the next page of the gamebook as structured data.",
				InputSchema: emitPageInputSchema,
			},
		},
		ToolChoice: &anthropicToolChoice{Type: "tool", Name: "emit_page"},
	}
	resp, err := a.postMessages(ctx, req)
	if err != nil {
		return domain.PageDraft{}, fmt.Errorf("llm.anthropic: NextPage: %w", err)
	}
	draft, err := parseToolUse(resp)
	if err != nil {
		return domain.PageDraft{}, fmt.Errorf("llm.anthropic: NextPage: %w", err)
	}
	if err := validateDraft(draft, in.Seq); err != nil {
		return domain.PageDraft{}, fmt.Errorf("llm.anthropic: NextPage: %w", err)
	}
	return draft, nil
}

func buildNextPageUserPrompt(in ports.StoryProviderInput) string {
	if in.Seq == 0 {
		return fmt.Sprintf("Topic: %s.\n\nOpen page 1 in media res -- the protagonist is already in motion, already facing something. Establish their immediate situation, an implicit want or fear, and close with 2-3 dilemmic choices, each promising a different KIND of trouble. Do NOT end the story yet; this is page 1 of a 4-8 page arc.", in.Topic)
	}
	var pacing string
	switch {
	case in.Seq >= 7:
		pacing = "PACING: this MUST be the final page. Set isEnding=true, return no choices, and pick the endingType that best fits how the arc has played out."
	case in.Seq >= 5:
		pacing = "PACING: bend toward resolution. Prefer ending here over continuing unless there is real unresolved tension. If ending, set isEnding=true, no choices, pick endingType."
	case in.Seq >= 3:
		pacing = "PACING: an ending is allowed only if this beat is genuinely a climax. Otherwise keep the story open with 2-3 choices."
	default:
		pacing = "PACING: do NOT end the story. This is too early. Continue with 2-3 choices; isEnding MUST be false."
	}
	return fmt.Sprintf(
		"Running summary so far: %s\n\nThe reader chose: %q.\n\nWrite page %d of a 4-8 page arc.\n\n%s",
		in.PriorSummary, in.ChosenText, in.Seq+1, pacing,
	)
}

func (a *Anthropic) postMessages(ctx context.Context, req anthropicRequest) (*anthropicResponse, error) {
	endpoint := a.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("x-api-key", a.APIKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := a.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		a.Log.Warn().Int("status", httpResp.StatusCode).Bytes("body", respBody).Msg("non-2xx from anthropic")
		if httpResp.StatusCode == http.StatusTooManyRequests || httpResp.StatusCode >= 500 {
			return nil, fmt.Errorf("status %d: %w", httpResp.StatusCode, ErrLLMTransient)
		}
		return nil, fmt.Errorf("status %d: %s", httpResp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed anthropicResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &parsed, nil
}

func parseToolUse(resp *anthropicResponse) (domain.PageDraft, error) {
	for _, block := range resp.Content {
		if block.Type != "tool_use" || block.Name != "emit_page" {
			continue
		}
		var input emitPageInput
		if err := json.Unmarshal(block.Input, &input); err != nil {
			return domain.PageDraft{}, fmt.Errorf("unmarshal tool input: %w", err)
		}
		return domain.PageDraft{
			Title:          strings.TrimSpace(input.Title),
			Narrative:      input.Narrative,
			ImagePrompt:    appendDepthSuffix(input.ImagePrompt),
			Choices:        input.Choices,
			IsEnding:       input.IsEnding,
			EndingType:     domain.EndingType(input.EndingType),
			RunningSummary: input.RunningSummary,
			Language:       strings.TrimSpace(input.Language),
		}, nil
	}
	return domain.PageDraft{}, errors.New("no tool_use block in response")
}

func appendDepthSuffix(prompt string) string {
	p := strings.TrimSpace(prompt)
	if p == "" {
		return p
	}
	if strings.Contains(strings.ToLower(p), "foreground, midground") {
		return p
	}
	p = strings.TrimRight(p, ".")
	return p + imagePromptDepthSuffix
}

func validateDraft(d domain.PageDraft, seq int) error {
	if strings.TrimSpace(d.Narrative) == "" {
		return errors.New("validate draft: narrative is empty")
	}
	if d.IsEnding {
		if seq < 3 {
			return fmt.Errorf("validate draft: ending too early at seq %d; minimum is 3 (page 4)", seq)
		}
		switch d.EndingType {
		case domain.EndingVictory, domain.EndingDefeat, domain.EndingTwist:
			return nil
		default:
			return fmt.Errorf("validate draft: invalid endingType %q", d.EndingType)
		}
	}
	if len(d.Choices) < 2 || len(d.Choices) > 3 {
		return fmt.Errorf("validate draft: expected 2-3 choices, got %d", len(d.Choices))
	}
	for i, c := range d.Choices {
		if strings.TrimSpace(c.Label) == "" {
			return fmt.Errorf("validate draft: choice %d has empty label", i)
		}
	}
	return nil
}
