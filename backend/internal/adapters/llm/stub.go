package llm

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

// Stub is a deterministic StoryProvider for local dev and E2E without an LLM account.
// It ends the story on the 4th page so the ending flow is exercised.
type Stub struct {
	calls atomic.Int32
}

func NewStub() *Stub { return &Stub{} }

func (s *Stub) StartStyle(_ context.Context, topic string) (domain.StylePrefix, error) {
	return domain.StylePrefix(fmt.Sprintf("ink-and-watercolor storybook illustration, muted palette, atmospheric, %s", topic)), nil
}

func (s *Stub) NextPage(_ context.Context, in ports.StoryProviderInput) (domain.PageDraft, error) {
	n := s.calls.Add(1)

	if in.Seq >= 3 {
		return domain.PageDraft{
			Narrative:      fmt.Sprintf("You chose %q. The tale settles into an unexpected calm, and the world holds its breath. Whatever comes next, you have already changed it. The book closes, leaving the last page warm under your hand.", in.ChosenText),
			ImagePrompt:    "quiet dusk over a still landscape, a single figure at the threshold of a doorway with light pouring through",
			Choices:        []domain.Choice{},
			IsEnding:       true,
			EndingType:     domain.EndingTwist,
			RunningSummary: "The story resolves in quiet surprise.",
		}, nil
	}

	narrative := "You stand at the edge of the unknown, the air tasting of salt and iron. Choices present themselves, each promising something different. The wind pulls at your coat; somewhere behind you, a distant bell rings once and falls silent. You can feel the weight of what is about to happen."
	if in.Seq > 0 {
		narrative = fmt.Sprintf("Having chosen %q, you find yourself in a new scene. The light is different now, slanting through dust and memory. Someone you do not yet know is watching. Page %d.\n\n%s", in.ChosenText, in.Seq+1, narrative)
	} else {
		narrative = fmt.Sprintf("Topic: %s.\n\n%s", in.Topic, narrative)
	}

	return domain.PageDraft{
		Narrative:   narrative,
		ImagePrompt: fmt.Sprintf("page %d illustration, a lone protagonist facing a choice, cinematic composition", in.Seq+1),
		Choices: []domain.Choice{
			{Label: "Step forward into the light"},
			{Label: "Turn back and listen"},
			{Label: "Follow the stranger"},
		},
		IsEnding:       false,
		RunningSummary: fmt.Sprintf("Page %d of a developing tale (call #%d).", in.Seq+1, n),
	}, nil
}
