package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

type Service struct {
	stories ports.StoryStore
	story   ports.StoryProvider
	images  ports.ImageProvider
	log     zerolog.Logger
	now     func() time.Time
	newID   func() string
}

func NewService(store ports.StoryStore, sp ports.StoryProvider, ip ports.ImageProvider, log zerolog.Logger) *Service {
	return &Service{
		stories: store,
		story:   sp,
		images:  ip,
		log:     log.With().Str("component", "app.service").Logger(),
		now:     func() time.Time { return time.Now().UTC() },
		newID:   func() string { return uuid.NewString() },
	}
}

func (s *Service) StartStory(ctx context.Context, in domain.StartStoryInput) (domain.StartStoryOutput, error) {
	style, err := s.story.StartStyle(ctx, in.Topic)
	if err != nil {
		return domain.StartStoryOutput{}, fmt.Errorf("app: start style: %w", err)
	}

	story := domain.Story{
		ID:          s.newID(),
		Topic:       in.Topic,
		StylePrefix: style,
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
	}
	if err := s.stories.Create(ctx, story); err != nil {
		return domain.StartStoryOutput{}, fmt.Errorf("app: create story: %w", err)
	}

	page, err := s.generatePage(ctx, story, "", "", 0)
	if err != nil {
		return domain.StartStoryOutput{}, err
	}

	return domain.StartStoryOutput{
		StoryID:     story.ID,
		StylePrefix: style,
		Page:        page,
	}, nil
}

func (s *Service) ProgressStory(ctx context.Context, in domain.ProgressInput) (domain.ProgressOutput, error) {
	story, err := s.stories.Get(ctx, in.StoryID)
	if err != nil {
		return domain.ProgressOutput{}, err
	}
	cur := story.Current()
	if cur == nil {
		return domain.ProgressOutput{}, fmt.Errorf("app: story has no pages")
	}
	if cur.IsEnding {
		return domain.ProgressOutput{}, domain.ErrStoryEnded
	}
	if in.ChoiceIndex < 0 || in.ChoiceIndex >= len(cur.Choices) {
		return domain.ProgressOutput{}, domain.ErrInvalidChoice
	}

	page, err := s.generatePage(ctx, story, cur.RunningSummary, cur.Choices[in.ChoiceIndex].Label, cur.Index+1)
	if err != nil {
		return domain.ProgressOutput{}, err
	}
	return domain.ProgressOutput{Page: page}, nil
}

func (s *Service) GetStory(ctx context.Context, id string) (domain.Story, error) {
	return s.stories.Get(ctx, id)
}

func (s *Service) GenerateImage(ctx context.Context, in domain.GenerateImageInput) (domain.GenerateImageOutput, error) {
	res, err := s.images.Generate(ctx, ports.ImageRequest{Prompt: in.Prompt, StylePrefix: in.StylePrefix})
	if err != nil {
		return domain.GenerateImageOutput{}, fmt.Errorf("app: generate image: %w", err)
	}
	return domain.GenerateImageOutput{URL: res.URL, Provider: res.Provider}, nil
}

// generatePage runs text generation serially, then offloads image generation in parallel.
// Image failure is non-fatal: the page is returned with an empty ImageURL.
func (s *Service) generatePage(ctx context.Context, story domain.Story, priorSummary, chosen string, seq int) (domain.Page, error) {
	draft, err := s.story.NextPage(ctx, ports.StoryProviderInput{
		Topic:        story.Topic,
		StylePrefix:  story.StylePrefix,
		PriorSummary: priorSummary,
		ChosenText:   chosen,
		Seq:          seq,
	})
	if err != nil {
		return domain.Page{}, fmt.Errorf("app: next page: %w", err)
	}

	page := domain.Page{
		Index:          seq,
		Narrative:      draft.Narrative,
		ImagePrompt:    draft.ImagePrompt,
		Choices:        draft.Choices,
		IsEnding:       draft.IsEnding,
		EndingType:     draft.EndingType,
		RunningSummary: draft.RunningSummary,
		CreatedAt:      s.now(),
	}

	g, gctx := errgroup.WithContext(ctx)
	var imgRes ports.ImageResult
	g.Go(func() error {
		r, err := s.images.Generate(gctx, ports.ImageRequest{Prompt: draft.ImagePrompt, StylePrefix: story.StylePrefix})
		imgRes = r
		return err
	})
	if err := g.Wait(); err != nil {
		s.log.Warn().Err(err).Str("story_id", story.ID).Int("seq", seq).Msg("image generation failed; continuing without image")
	} else {
		page.ImageURL = imgRes.URL
		page.ImageProvider = imgRes.Provider
	}

	if err := s.stories.AppendPage(ctx, story.ID, page); err != nil {
		if errors.Is(err, domain.ErrStoryNotFound) {
			return domain.Page{}, err
		}
		return domain.Page{}, fmt.Errorf("app: append page: %w", err)
	}
	return page, nil
}
