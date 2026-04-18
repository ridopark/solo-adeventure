package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

type capturedRequest struct {
	headers http.Header
	body    []byte
}

func newTestServer(t *testing.T, status int, respBody string, captured *capturedRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if captured != nil {
			captured.headers = r.Header.Clone()
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			captured.body = b
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, respBody)
	}))
}

func newAnthropicForTest(t *testing.T, endpoint string) *Anthropic {
	t.Helper()
	a := NewAnthropic("test-key", "claude-sonnet-4-6", &http.Client{}, zerolog.Nop())
	a.Endpoint = endpoint
	return a
}

func toolUseResponse(t *testing.T, input emitPageInput) string {
	t.Helper()
	inp, err := json.Marshal(input)
	require.NoError(t, err)
	resp := anthropicResponse{
		ID:   "msg_test",
		Role: "assistant",
		Content: []anthropicContentBlock{
			{Type: "tool_use", ID: "toolu_1", Name: "emit_page", Input: inp},
		},
		StopReason: "tool_use",
	}
	b, err := json.Marshal(resp)
	require.NoError(t, err)
	return string(b)
}

func validEmitPage() emitPageInput {
	return emitPageInput{
		Narrative:   "You stand at the edge of a cliff as the wind roars in your ears.",
		ImagePrompt: "A lone traveler at a cliff edge under stormy skies.",
		Choices: []domain.Choice{
			{Label: "Leap"},
			{Label: "Turn back"},
		},
		IsEnding:       false,
		EndingType:     "",
		RunningSummary: "Traveler arrives at a cliff.",
	}
}

func TestNextPage_HappyPath_FirstPage(t *testing.T) {
	captured := &capturedRequest{}
	srv := newTestServer(t, 200, toolUseResponse(t, validEmitPage()), captured)
	defer srv.Close()

	a := newAnthropicForTest(t, srv.URL)

	draft, err := a.NextPage(context.Background(), ports.StoryProviderInput{
		Topic: "lost ancient temple",
		Seq:   0,
	})
	require.NoError(t, err)

	assert.Equal(t, "You stand at the edge of a cliff as the wind roars in your ears.", draft.Narrative)
	assert.Contains(t, draft.ImagePrompt, "A lone traveler at a cliff edge under stormy skies")
	assert.Contains(t, draft.ImagePrompt, "foreground, midground, and background")
	assert.Len(t, draft.Choices, 2)
	assert.Equal(t, "Leap", draft.Choices[0].Label)
	assert.False(t, draft.IsEnding)
	assert.Equal(t, "Traveler arrives at a cliff.", draft.RunningSummary)

	assert.Equal(t, "test-key", captured.headers.Get("x-api-key"))
	assert.Equal(t, "2023-06-01", captured.headers.Get("anthropic-version"))
	assert.Equal(t, "application/json", captured.headers.Get("Content-Type"))

	var req anthropicRequest
	require.NoError(t, json.Unmarshal(captured.body, &req))
	assert.Equal(t, "claude-sonnet-4-6", req.Model)
	require.NotNil(t, req.ToolChoice)
	assert.Equal(t, "tool", req.ToolChoice.Type)
	assert.Equal(t, "emit_page", req.ToolChoice.Name)
	require.Len(t, req.Tools, 1)
	assert.Equal(t, "emit_page", req.Tools[0].Name)
	require.Len(t, req.Messages, 1)
	assert.Contains(t, req.Messages[0].Content, "lost ancient temple")
	assert.Contains(t, req.Messages[0].Content, "page 1")
}

func TestNextPage_HappyPath_SubsequentPage(t *testing.T) {
	captured := &capturedRequest{}
	srv := newTestServer(t, 200, toolUseResponse(t, validEmitPage()), captured)
	defer srv.Close()

	a := newAnthropicForTest(t, srv.URL)

	in := ports.StoryProviderInput{
		Topic:        "lost ancient temple",
		PriorSummary: "You discovered a hidden map and descended into catacombs.",
		ChosenText:   "Light the torch",
		Seq:          2,
	}
	_, err := a.NextPage(context.Background(), in)
	require.NoError(t, err)

	var req anthropicRequest
	require.NoError(t, json.Unmarshal(captured.body, &req))
	require.Len(t, req.Messages, 1)
	content := req.Messages[0].Content
	assert.Contains(t, content, "You discovered a hidden map and descended into catacombs.")
	assert.Contains(t, content, "Light the torch")
	assert.Contains(t, content, "page 3")
}

func TestNextPage_MissingToolUse(t *testing.T) {
	resp := anthropicResponse{
		ID:   "msg_x",
		Role: "assistant",
		Content: []anthropicContentBlock{
			{Type: "text", Text: "sorry, I cannot"},
		},
		StopReason: "end_turn",
	}
	body, err := json.Marshal(resp)
	require.NoError(t, err)

	srv := newTestServer(t, 200, string(body), nil)
	defer srv.Close()

	a := newAnthropicForTest(t, srv.URL)

	_, err = a.NextPage(context.Background(), ports.StoryProviderInput{Topic: "x", Seq: 0})
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrLLMTransient))
	assert.Contains(t, err.Error(), "llm.anthropic: NextPage:")
}

func TestNextPage_StatusCodes(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		transient bool
	}{
		{"rate limited", 429, true},
		{"server error", 500, true},
		{"bad gateway", 502, true},
		{"bad request", 400, false},
		{"unauthorized", 401, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer(t, tc.status, `{"error":"boom"}`, nil)
			defer srv.Close()

			a := newAnthropicForTest(t, srv.URL)
			_, err := a.NextPage(context.Background(), ports.StoryProviderInput{Topic: "x", Seq: 0})
			require.Error(t, err)
			assert.Equal(t, tc.transient, errors.Is(err, ErrLLMTransient), "transient classification mismatch")
		})
	}
}

func TestNextPage_ValidationErrors(t *testing.T) {
	missingNarrative := validEmitPage()
	missingNarrative.Narrative = ""

	oneChoice := validEmitPage()
	oneChoice.Choices = []domain.Choice{{Label: "solo"}}

	emptyChoiceLabel := validEmitPage()
	emptyChoiceLabel.Choices = []domain.Choice{{Label: "a"}, {Label: ""}}

	endingInvalidType := validEmitPage()
	endingInvalidType.IsEnding = true
	endingInvalidType.Choices = nil
	endingInvalidType.EndingType = "bogus"

	cases := []struct {
		name  string
		input emitPageInput
	}{
		{"missing narrative", missingNarrative},
		{"only one choice", oneChoice},
		{"empty choice label", emptyChoiceLabel},
		{"ending with invalid endingType", endingInvalidType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newTestServer(t, 200, toolUseResponse(t, tc.input), nil)
			defer srv.Close()

			a := newAnthropicForTest(t, srv.URL)
			_, err := a.NextPage(context.Background(), ports.StoryProviderInput{Topic: "x", Seq: 0})
			require.Error(t, err)
			assert.False(t, errors.Is(err, ErrLLMTransient))
			assert.Contains(t, err.Error(), "validate draft")
		})
	}
}

func TestNextPage_EndingVictoryOK(t *testing.T) {
	input := emitPageInput{
		Narrative:      "You plant the flag; the mountain is yours.",
		ImagePrompt:    "A climber on a snowy summit, flag planted.",
		Choices:        nil,
		IsEnding:       true,
		EndingType:     "victory",
		RunningSummary: "Climber reaches the summit after many trials.",
	}
	srv := newTestServer(t, 200, toolUseResponse(t, input), nil)
	defer srv.Close()

	a := newAnthropicForTest(t, srv.URL)
	draft, err := a.NextPage(context.Background(), ports.StoryProviderInput{Topic: "mountain", Seq: 5})
	require.NoError(t, err)
	assert.True(t, draft.IsEnding)
	assert.Equal(t, domain.EndingVictory, draft.EndingType)
	assert.Empty(t, draft.Choices)
}

func TestStartStyle_HappyPath(t *testing.T) {
	resp := anthropicResponse{
		ID:   "msg_style",
		Role: "assistant",
		Content: []anthropicContentBlock{
			{Type: "text", Text: "  painterly watercolor with soft edges, warm muted palette  \n"},
		},
		StopReason: "end_turn",
	}
	body, err := json.Marshal(resp)
	require.NoError(t, err)

	captured := &capturedRequest{}
	srv := newTestServer(t, 200, string(body), captured)
	defer srv.Close()

	a := newAnthropicForTest(t, srv.URL)
	style, err := a.StartStyle(context.Background(), "mountain climbing")
	require.NoError(t, err)
	assert.Equal(t, domain.StylePrefix("painterly watercolor with soft edges, warm muted palette"), style)

	var req anthropicRequest
	require.NoError(t, json.Unmarshal(captured.body, &req))
	assert.Empty(t, req.Tools, "StartStyle must not include tools")
	assert.Nil(t, req.ToolChoice, "StartStyle must not include tool_choice")
	require.Len(t, req.Messages, 1)
	assert.Contains(t, req.Messages[0].Content, "mountain climbing")
	assert.True(t, strings.HasPrefix(req.System, "You produce short illustrator style descriptors"))
}

func TestStartStyle_NoTextBlock(t *testing.T) {
	resp := anthropicResponse{
		ID:      "msg_empty",
		Role:    "assistant",
		Content: []anthropicContentBlock{},
	}
	body, err := json.Marshal(resp)
	require.NoError(t, err)

	srv := newTestServer(t, 200, string(body), nil)
	defer srv.Close()

	a := newAnthropicForTest(t, srv.URL)
	_, err = a.StartStyle(context.Background(), "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "llm.anthropic: StartStyle:")
}
