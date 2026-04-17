package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ridopark/solo-adeventure/backend/internal/adapters/store"
	"github.com/ridopark/solo-adeventure/backend/internal/app"
	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

type stubStory struct {
	style  domain.StylePrefix
	drafts []domain.PageDraft
	calls  int
}

func (s *stubStory) StartStyle(ctx context.Context, topic string) (domain.StylePrefix, error) {
	return s.style, nil
}
func (s *stubStory) NextPage(ctx context.Context, in ports.StoryProviderInput) (domain.PageDraft, error) {
	d := s.drafts[s.calls]
	s.calls++
	return d, nil
}

type stubImage struct{}

func (stubImage) Generate(ctx context.Context, req ports.ImageRequest) (ports.ImageResult, error) {
	return ports.ImageResult{URL: "https://img/1.png", Provider: "stub"}, nil
}

func draftPage(narrative string, choices int) domain.PageDraft {
	cs := make([]domain.Choice, choices)
	for i := range cs {
		cs[i] = domain.Choice{Label: "x"}
	}
	return domain.PageDraft{Narrative: narrative, ImagePrompt: "i", Choices: cs, RunningSummary: "s"}
}

func newServer(t *testing.T, drafts ...domain.PageDraft) *httptest.Server {
	t.Helper()
	svc := app.NewService(
		store.NewMemory(),
		&stubStory{style: "style-x", drafts: drafts},
		stubImage{},
		zerolog.Nop(),
	)
	handler := NewRouter(svc, zerolog.Nop(), "*")
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, r)
	require.NoError(t, err)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestRouter_Health(t *testing.T) {
	srv := newServer(t)
	code, body := do(t, http.MethodGet, srv.URL+"/health", "")
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `"ok"`)
}

func TestRouter_StartStory_201(t *testing.T) {
	srv := newServer(t, draftPage("opening", 2))
	code, body := do(t, http.MethodPost, srv.URL+"/stories", `{"topic":"lighthouse"}`)
	assert.Equal(t, http.StatusCreated, code)

	var resp domain.StartStoryOutput
	require.NoError(t, json.Unmarshal([]byte(body), &resp))
	assert.NotEmpty(t, resp.StoryID)
	assert.Equal(t, domain.StylePrefix("style-x"), resp.StylePrefix)
	assert.Equal(t, "opening", resp.Page.Narrative)
}

func TestRouter_StartStory_BadJSON(t *testing.T) {
	srv := newServer(t)
	code, _ := do(t, http.MethodPost, srv.URL+"/stories", `not json`)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestRouter_StartStory_EmptyTopic(t *testing.T) {
	srv := newServer(t)
	code, body := do(t, http.MethodPost, srv.URL+"/stories", `{"topic":""}`)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, body, "topic required")
}

func TestRouter_GetStory_404(t *testing.T) {
	srv := newServer(t)
	code, _ := do(t, http.MethodGet, srv.URL+"/stories/nope", "")
	assert.Equal(t, http.StatusNotFound, code)
}

func TestRouter_ChooseStory_FullFlow(t *testing.T) {
	srv := newServer(t, draftPage("first", 2), draftPage("second", 2))

	code, body := do(t, http.MethodPost, srv.URL+"/stories", `{"topic":"x"}`)
	require.Equal(t, http.StatusCreated, code)
	var start domain.StartStoryOutput
	require.NoError(t, json.Unmarshal([]byte(body), &start))

	code, body = do(t, http.MethodPost, srv.URL+"/stories/"+start.StoryID+"/choose", `{"choiceIndex":0}`)
	assert.Equal(t, http.StatusOK, code)
	var prog domain.ProgressOutput
	require.NoError(t, json.Unmarshal([]byte(body), &prog))
	assert.Equal(t, "second", prog.Page.Narrative)

	code, body = do(t, http.MethodGet, srv.URL+"/stories/"+start.StoryID, "")
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, body, `"pages"`)
}

func TestRouter_ChooseStory_InvalidChoice400(t *testing.T) {
	srv := newServer(t, draftPage("first", 2))
	code, body := do(t, http.MethodPost, srv.URL+"/stories", `{"topic":"x"}`)
	require.Equal(t, http.StatusCreated, code)
	var start domain.StartStoryOutput
	require.NoError(t, json.Unmarshal([]byte(body), &start))

	code, _ = do(t, http.MethodPost, srv.URL+"/stories/"+start.StoryID+"/choose", `{"choiceIndex":99}`)
	assert.Equal(t, http.StatusBadRequest, code)
}

func TestRouter_CORSHeadersAndPreflight(t *testing.T) {
	srv := newServer(t)

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/stories", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))

	code, _ := do(t, http.MethodGet, srv.URL+"/health", "")
	assert.Equal(t, http.StatusOK, code)
}

func TestRouter_InvalidJSON_Stories(t *testing.T) {
	srv := newServer(t)
	code, _ := do(t, http.MethodPost, srv.URL+"/stories", `{`)
	assert.Equal(t, http.StatusBadRequest, code)
}

// silence unused import lint when testify/require not referenced directly above
var _ = bytes.NewReader
