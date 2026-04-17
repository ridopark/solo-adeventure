package app

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ridopark/solo-adeventure/backend/internal/adapters/store"
	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

type fakeStoryProvider struct {
	style    domain.StylePrefix
	styleErr error
	drafts   []domain.PageDraft
	err      error
	calls    int
}

func (f *fakeStoryProvider) StartStyle(ctx context.Context, topic string) (domain.StylePrefix, error) {
	return f.style, f.styleErr
}
func (f *fakeStoryProvider) NextPage(ctx context.Context, in ports.StoryProviderInput) (domain.PageDraft, error) {
	if f.err != nil {
		return domain.PageDraft{}, f.err
	}
	d := f.drafts[f.calls]
	f.calls++
	return d, nil
}

type fakeImageProvider struct {
	res     ports.ImageResult
	err     error
	invoked int
}

func (f *fakeImageProvider) Generate(ctx context.Context, req ports.ImageRequest) (ports.ImageResult, error) {
	f.invoked++
	return f.res, f.err
}

func draft(narrative string, choices int) domain.PageDraft {
	cs := make([]domain.Choice, choices)
	for i := range cs {
		cs[i] = domain.Choice{Label: "go"}
	}
	return domain.PageDraft{
		Narrative:      narrative,
		ImagePrompt:    "a scene",
		Choices:        cs,
		RunningSummary: "summary",
	}
}

type capturingNotifier struct {
	events []ports.NotifyEvent
}

func (c *capturingNotifier) Notify(ev ports.NotifyEvent) { c.events = append(c.events, ev) }

func newSvc(sp ports.StoryProvider, ip ports.ImageProvider) *Service {
	return NewService(store.NewMemory(), sp, ip, &capturingNotifier{}, zerolog.Nop())
}

func TestService_StartStory_HappyPath(t *testing.T) {
	sp := &fakeStoryProvider{style: "watercolor", drafts: []domain.PageDraft{draft("opening scene", 2)}}
	ip := &fakeImageProvider{res: ports.ImageResult{URL: "https://img/1.png", Provider: "together"}}

	svc := newSvc(sp, ip)
	out, err := svc.StartStory(context.Background(), domain.StartStoryInput{Topic: "lighthouse"})

	require.NoError(t, err)
	assert.NotEmpty(t, out.StoryID)
	assert.Equal(t, domain.StylePrefix("watercolor"), out.StylePrefix)
	assert.Equal(t, "opening scene", out.Page.Narrative)
	assert.Equal(t, "https://img/1.png", out.Page.ImageURL)
	assert.Equal(t, "together", out.Page.ImageProvider)
	assert.Len(t, out.Page.Choices, 2)
	assert.Equal(t, 1, ip.invoked)
}

func TestService_StartStory_ImageFailure_StillReturnsPage(t *testing.T) {
	sp := &fakeStoryProvider{style: "ink", drafts: []domain.PageDraft{draft("page one", 2)}}
	ip := &fakeImageProvider{err: errors.New("boom")}

	svc := newSvc(sp, ip)
	out, err := svc.StartStory(context.Background(), domain.StartStoryInput{Topic: "x"})

	require.NoError(t, err, "image failure must not fail the page")
	assert.Equal(t, "", out.Page.ImageURL)
	assert.Equal(t, "", out.Page.ImageProvider)
	assert.Equal(t, "page one", out.Page.Narrative)
}

func TestService_StartStory_TextFailure_ReturnsWrappedError(t *testing.T) {
	sp := &fakeStoryProvider{style: "s", err: errors.New("anthropic 500")}
	ip := &fakeImageProvider{}

	svc := newSvc(sp, ip)
	_, err := svc.StartStory(context.Background(), domain.StartStoryInput{Topic: "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app: next page")
}

func TestService_ProgressStory_HappyPath(t *testing.T) {
	sp := &fakeStoryProvider{
		style:  "ink",
		drafts: []domain.PageDraft{draft("first", 2), draft("second", 2)},
	}
	ip := &fakeImageProvider{res: ports.ImageResult{URL: "u", Provider: "fal"}}

	svc := newSvc(sp, ip)
	start, err := svc.StartStory(context.Background(), domain.StartStoryInput{Topic: "t"})
	require.NoError(t, err)

	next, err := svc.ProgressStory(context.Background(), domain.ProgressInput{StoryID: start.StoryID, ChoiceIndex: 0})
	require.NoError(t, err)
	assert.Equal(t, "second", next.Page.Narrative)
	assert.Equal(t, 1, next.Page.Index)
}

func TestService_ProgressStory_InvalidChoice(t *testing.T) {
	sp := &fakeStoryProvider{style: "s", drafts: []domain.PageDraft{draft("only page", 2)}}
	ip := &fakeImageProvider{res: ports.ImageResult{URL: "u", Provider: "fal"}}

	svc := newSvc(sp, ip)
	start, err := svc.StartStory(context.Background(), domain.StartStoryInput{Topic: "t"})
	require.NoError(t, err)

	_, err = svc.ProgressStory(context.Background(), domain.ProgressInput{StoryID: start.StoryID, ChoiceIndex: 99})
	assert.True(t, errors.Is(err, domain.ErrInvalidChoice))
}

func TestService_ProgressStory_StoryNotFound(t *testing.T) {
	svc := newSvc(&fakeStoryProvider{}, &fakeImageProvider{})
	_, err := svc.ProgressStory(context.Background(), domain.ProgressInput{StoryID: "does-not-exist", ChoiceIndex: 0})
	assert.True(t, errors.Is(err, domain.ErrStoryNotFound))
}

func TestService_ProgressStory_EndedStoryRejected(t *testing.T) {
	endingDraft := domain.PageDraft{
		Narrative:      "the end",
		ImagePrompt:    "sunset",
		IsEnding:       true,
		EndingType:     domain.EndingVictory,
		RunningSummary: "done",
	}
	sp := &fakeStoryProvider{style: "s", drafts: []domain.PageDraft{endingDraft}}
	ip := &fakeImageProvider{}

	svc := newSvc(sp, ip)
	start, err := svc.StartStory(context.Background(), domain.StartStoryInput{Topic: "t"})
	require.NoError(t, err)
	require.True(t, start.Page.IsEnding)

	_, err = svc.ProgressStory(context.Background(), domain.ProgressInput{StoryID: start.StoryID, ChoiceIndex: 0})
	assert.True(t, errors.Is(err, domain.ErrStoryEnded))
}

func TestService_GetStory(t *testing.T) {
	sp := &fakeStoryProvider{style: "s", drafts: []domain.PageDraft{draft("p", 2)}}
	ip := &fakeImageProvider{}

	svc := newSvc(sp, ip)
	start, err := svc.StartStory(context.Background(), domain.StartStoryInput{Topic: "t"})
	require.NoError(t, err)

	got, err := svc.GetStory(context.Background(), start.StoryID)
	require.NoError(t, err)
	assert.Equal(t, start.StoryID, got.ID)
	assert.Len(t, got.Pages, 1)
}

func TestService_GenerateImage_Passthrough(t *testing.T) {
	ip := &fakeImageProvider{res: ports.ImageResult{URL: "u", Provider: "together"}}
	svc := newSvc(&fakeStoryProvider{}, ip)

	out, err := svc.GenerateImage(context.Background(), domain.GenerateImageInput{Prompt: "p"})
	require.NoError(t, err)
	assert.Equal(t, "u", out.URL)
	assert.Equal(t, "together", out.Provider)
}
