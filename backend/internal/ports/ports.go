package ports

import (
	"context"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

// StoryProviderInput is the context a StoryProvider needs to produce the next page.
// On the first page PriorSummary and ChosenText are empty.
type StoryProviderInput struct {
	Topic        string
	StylePrefix  domain.StylePrefix
	PriorSummary string
	ChosenText   string
	Seq          int
}

type StoryProvider interface {
	StartStyle(ctx context.Context, topic string) (domain.StylePrefix, error)
	NextPage(ctx context.Context, in StoryProviderInput) (domain.PageDraft, error)
}

type ImageRequest struct {
	Prompt      string
	StylePrefix domain.StylePrefix
}

type ImageResult struct {
	URL      string
	Provider string
}

type ImageProvider interface {
	Generate(ctx context.Context, req ImageRequest) (ImageResult, error)
}

type StoryStore interface {
	Create(ctx context.Context, s domain.Story) error
	Get(ctx context.Context, id string) (domain.Story, error)
	AppendPage(ctx context.Context, storyID string, p domain.Page) error
}
