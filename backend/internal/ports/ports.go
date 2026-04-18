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
	ListByUser(ctx context.Context, userID string, limit int) ([]domain.Story, error)
	AttachUser(ctx context.Context, storyID, userID string) error
	UpdatePageAudio(ctx context.Context, storyID string, idx int, audioURL string) error
}

type TTSRequest struct {
	Text  string
	Voice string
}

type TTSResult struct {
	Audio  []byte
	Format string
}

type TTSProvider interface {
	Synthesize(ctx context.Context, req TTSRequest) (TTSResult, error)
}

type UserStore interface {
	GetByGoogleSub(ctx context.Context, sub string) (domain.User, error)
	Create(ctx context.Context, u domain.User) error
	Get(ctx context.Context, id string) (domain.User, error)
}

type SessionStore interface {
	Create(ctx context.Context, s domain.Session) error
	Get(ctx context.Context, id string) (domain.Session, error)
	Delete(ctx context.Context, id string) error
}

type OAuthProvider interface {
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (domain.OAuthUserInfo, error)
}

type NotifyKind string

const (
	NotifyTopicSubmitted NotifyKind = "topic_submitted"
	NotifyPageGenerated  NotifyKind = "page_generated"
	NotifyImageGenerated NotifyKind = "image_generated"
	NotifyChoiceMade     NotifyKind = "choice_made"
	NotifyVisitStarted   NotifyKind = "visit_started"
)

type NotifyField struct {
	Name  string
	Value string
}

type NotifyEvent struct {
	Kind          NotifyKind
	StoryID       string
	Title         string
	Message       string
	ImageURL      string
	Fields        []NotifyField
	UserName      string
	UserAvatarURL string
}

type Notifier interface {
	Notify(event NotifyEvent)
}
