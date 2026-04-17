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

const systemPromptNextPage = "You are the author of a choose-your-own-adventure gamebook. Write in second person, present tense. Each page is a single vignette 150-250 words long. End every page with 2-3 distinct, consequential choices -- unless the story has concluded, in which case return no choices, set isEnding=true, and pick an endingType (victory/defeat/twist). You must always invoke the emit_page tool; never respond in plain text."

const systemPromptStyle = "You produce short illustrator style descriptors (8-15 words) for storybook art. Output ONLY the descriptor; no preamble, no quotes."

var emitPageInputSchema = json.RawMessage(`{
  "type": "object",
  "required": ["narrative", "imagePrompt", "choices", "isEnding", "runningSummary"],
  "properties": {
    "narrative":      { "type": "string", "description": "150-250 words, second person, present tense" },
    "imagePrompt":    { "type": "string", "description": "visual-only description of this scene; no dialogue, no text" },
    "choices":        { "type": "array", "minItems": 0, "maxItems": 3, "items": { "type": "object", "required": ["label"], "properties": { "label": { "type": "string" } } } },
    "isEnding":       { "type": "boolean" },
    "endingType":     { "type": "string", "enum": ["victory", "defeat", "twist", ""] },
    "runningSummary": { "type": "string", "description": "2-3 sentence summary of the whole story so far, used as context for the next page" }
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
	Narrative      string          `json:"narrative"`
	ImagePrompt    string          `json:"imagePrompt"`
	Choices        []domain.Choice `json:"choices"`
	IsEnding       bool            `json:"isEnding"`
	EndingType     string          `json:"endingType"`
	RunningSummary string          `json:"runningSummary"`
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
	if err := validateDraft(draft); err != nil {
		return domain.PageDraft{}, fmt.Errorf("llm.anthropic: NextPage: %w", err)
	}
	return draft, nil
}

func buildNextPageUserPrompt(in ports.StoryProviderInput) string {
	if in.Seq == 0 {
		return fmt.Sprintf("Topic: %s. Open the story on page 1.", in.Topic)
	}
	return fmt.Sprintf("Running summary so far: %s\n\nThe reader chose: \"%s\".\n\nWrite page %d. End the story naturally if this is a satisfying conclusion.", in.PriorSummary, in.ChosenText, in.Seq+1)
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
			Narrative:      input.Narrative,
			ImagePrompt:    input.ImagePrompt,
			Choices:        input.Choices,
			IsEnding:       input.IsEnding,
			EndingType:     domain.EndingType(input.EndingType),
			RunningSummary: input.RunningSummary,
		}, nil
	}
	return domain.PageDraft{}, errors.New("no tool_use block in response")
}

func validateDraft(d domain.PageDraft) error {
	if strings.TrimSpace(d.Narrative) == "" {
		return errors.New("validate draft: narrative is empty")
	}
	if d.IsEnding {
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
