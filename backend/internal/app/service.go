package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
	"github.com/ridopark/solo-adeventure/backend/internal/ports"
)

type Service struct {
	stories    ports.StoryStore
	story      ports.StoryProvider
	images     ports.ImageProvider
	notifier   ports.Notifier
	users      ports.UserStore
	sessions   ports.SessionStore
	oauth      ports.OAuthProvider
	tts        ports.TTSProvider
	audioDir   string
	audioBase  string
	log        zerolog.Logger
	now        func() time.Time
	newID      func() string
}

func NewService(store ports.StoryStore, sp ports.StoryProvider, ip ports.ImageProvider, notifier ports.Notifier, log zerolog.Logger) *Service {
	return &Service{
		stories:  store,
		story:    sp,
		images:   ip,
		notifier: notifier,
		log:      log.With().Str("component", "app.service").Logger(),
		now:      func() time.Time { return time.Now().UTC() },
		newID:    func() string { return uuid.NewString() },
	}
}

// WithAuth wires optional auth collaborators. Called after NewService when the
// process has SQLite-backed user/session stores and a Google OAuth client.
func (s *Service) WithAuth(users ports.UserStore, sessions ports.SessionStore, oauth ports.OAuthProvider) *Service {
	s.users = users
	s.sessions = sessions
	s.oauth = oauth
	return s
}

// WithTTS wires the optional narration sidecar. audioDir is where cached MP3s
// live on disk; audioBase is the URL prefix the frontend uses to fetch them
// (must match the static file server mount).
func (s *Service) WithTTS(tts ports.TTSProvider, audioDir, audioBase string) *Service {
	s.tts = tts
	s.audioDir = audioDir
	s.audioBase = audioBase
	return s
}

// currentUserAttribution returns name+avatar for the authenticated caller, or
// empty strings if the request is anonymous or user lookup fails. Used to tag
// NotifyEvents with a human-readable identity.
func (s *Service) currentUserAttribution(ctx context.Context) (name, avatar string) {
	uid := CurrentUserID(ctx)
	if uid == "" || s.users == nil {
		return "", ""
	}
	u, err := s.users.Get(ctx, uid)
	if err != nil {
		return "", ""
	}
	if u.Name != "" {
		return u.Name, u.AvatarURL
	}
	return u.Email, u.AvatarURL
}

func (s *Service) StartStory(ctx context.Context, in domain.StartStoryInput) (domain.StartStoryOutput, error) {
	if err := domain.ValidateTopic(in.Topic); err != nil {
		return domain.StartStoryOutput{}, err
	}
	style, err := s.story.StartStyle(ctx, in.Topic)
	if err != nil {
		return domain.StartStoryOutput{}, fmt.Errorf("app: start style: %w", err)
	}

	story := domain.Story{
		ID:          s.newID(),
		UserID:      CurrentUserID(ctx),
		Topic:       in.Topic,
		StylePrefix: style,
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
	}
	if err := s.stories.Create(ctx, story); err != nil {
		return domain.StartStoryOutput{}, fmt.Errorf("app: create story: %w", err)
	}

	userName, userAvatar := s.currentUserAttribution(ctx)
	s.notifier.Notify(ports.NotifyEvent{
		Kind:          ports.NotifyTopicSubmitted,
		StoryID:       story.ID,
		Title:         "New adventure started",
		Message:       in.Topic,
		UserName:      userName,
		UserAvatarURL: userAvatar,
		Fields: []ports.NotifyField{
			{Name: "style", Value: string(style)},
		},
	})

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
	userID := CurrentUserID(ctx)
	if userID == "" {
		return domain.ProgressOutput{}, domain.ErrUnauthorized
	}
	story, err := s.stories.Get(ctx, in.StoryID)
	if err != nil {
		return domain.ProgressOutput{}, err
	}
	if story.UserID != "" && story.UserID != userID {
		return domain.ProgressOutput{}, domain.ErrForbidden
	}
	if story.UserID == "" {
		if err := s.stories.AttachUser(ctx, story.ID, userID); err == nil {
			story.UserID = userID
		}
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

	chosen := cur.Choices[in.ChoiceIndex].Label
	userName, userAvatar := s.currentUserAttribution(ctx)
	s.notifier.Notify(ports.NotifyEvent{
		Kind:          ports.NotifyChoiceMade,
		StoryID:       story.ID,
		Title:         fmt.Sprintf("Choice on page %d", cur.Index+1),
		Message:       chosen,
		UserName:      userName,
		UserAvatarURL: userAvatar,
		Fields: []ports.NotifyField{
			{Name: "choiceIndex", Value: fmt.Sprintf("%d", in.ChoiceIndex)},
		},
	})

	page, err := s.generatePage(ctx, story, cur.RunningSummary, chosen, cur.Index+1)
	if err != nil {
		return domain.ProgressOutput{}, err
	}
	return domain.ProgressOutput{Page: page}, nil
}

func (s *Service) GetStory(ctx context.Context, id string) (domain.Story, error) {
	return s.stories.Get(ctx, id)
}

func (s *Service) RecordVisit(ctx context.Context, in domain.VisitInput) {
	path := in.Path
	if path == "" {
		path = "/"
	}
	fields := []ports.NotifyField{
		{Name: "path", Value: path},
	}
	if in.Referrer != "" {
		fields = append(fields, ports.NotifyField{Name: "referrer", Value: in.Referrer})
	}
	if in.UserAgent != "" {
		fields = append(fields, ports.NotifyField{Name: "user agent", Value: truncate(in.UserAgent, 180)})
	}
	userName, userAvatar := s.currentUserAttribution(ctx)
	s.notifier.Notify(ports.NotifyEvent{
		Kind:          ports.NotifyVisitStarted,
		Title:         "A reader arrived",
		Message:       "",
		UserName:      userName,
		UserAvatarURL: userAvatar,
		Fields:        fields,
	})
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

	pageFields := []ports.NotifyField{
		{Name: "page", Value: fmt.Sprintf("%d", seq+1)},
	}
	if page.IsEnding {
		pageFields = append(pageFields, ports.NotifyField{Name: "ending", Value: string(page.EndingType)})
	} else {
		pageFields = append(pageFields, ports.NotifyField{Name: "choices", Value: fmt.Sprintf("%d", len(page.Choices))})
	}
	userName, userAvatar := s.currentUserAttribution(ctx)
	s.notifier.Notify(ports.NotifyEvent{
		Kind:          ports.NotifyPageGenerated,
		StoryID:       story.ID,
		Title:         fmt.Sprintf("Page %d generated", seq+1),
		Message:       truncate(page.Narrative, 300),
		UserName:      userName,
		UserAvatarURL: userAvatar,
		Fields:        pageFields,
	})

	g, gctx := errgroup.WithContext(ctx)
	var imgRes ports.ImageResult
	if err := domain.ValidateImagePrompt(draft.ImagePrompt); err != nil {
		s.log.Warn().Str("story_id", story.ID).Int("seq", seq).Msg("image prompt rejected by safety filter; skipping image")
	} else {
		g.Go(func() error {
			r, err := s.images.Generate(gctx, ports.ImageRequest{Prompt: draft.ImagePrompt, StylePrefix: story.StylePrefix})
			imgRes = r
			return err
		})
	}
	if err := g.Wait(); err != nil {
		s.log.Warn().Err(err).Str("story_id", story.ID).Int("seq", seq).Msg("image generation failed; continuing without image")
	} else if imgRes.URL != "" {
		page.ImageURL = imgRes.URL
		page.ImageProvider = imgRes.Provider
		s.notifier.Notify(ports.NotifyEvent{
			Kind:          ports.NotifyImageGenerated,
			StoryID:       story.ID,
			Title:         fmt.Sprintf("Page %d illustration", seq+1),
			Message:       truncate(draft.ImagePrompt, 200),
			ImageURL:      page.ImageURL,
			UserName:      userName,
			UserAvatarURL: userAvatar,
			Fields: []ports.NotifyField{
				{Name: "provider", Value: page.ImageProvider},
			},
		})
	}

	if err := s.stories.AppendPage(ctx, story.ID, page); err != nil {
		if errors.Is(err, domain.ErrStoryNotFound) {
			return domain.Page{}, err
		}
		return domain.Page{}, fmt.Errorf("app: append page: %w", err)
	}
	return page, nil
}

// GenerateSpeech returns a cached audio URL for a page narrative, synthesizing
// it via the TTS sidecar on first call. Narrative text never mutates once a
// page is written, so the first call pays the TTS cost and all subsequent
// calls are free disk reads.
func (s *Service) GenerateSpeech(ctx context.Context, storyID string, seq int) (string, error) {
	if s.tts == nil || s.audioDir == "" {
		return "", domain.ErrTTSUnavailable
	}
	story, err := s.stories.Get(ctx, storyID)
	if err != nil {
		return "", err
	}
	if seq < 0 || seq >= len(story.Pages) {
		return "", domain.ErrPageNotFound
	}
	page := story.Pages[seq]
	if page.AudioURL != "" {
		return page.AudioURL, nil
	}

	res, err := s.tts.Synthesize(ctx, ports.TTSRequest{Text: page.Narrative})
	if err != nil {
		s.log.Warn().Err(err).Str("story_id", storyID).Int("seq", seq).Msg("tts synthesize failed")
		return "", domain.ErrTTSUnavailable
	}

	filename := fmt.Sprintf("%s-%d.mp3", storyID, seq)
	if err := os.MkdirAll(s.audioDir, 0o755); err != nil {
		return "", fmt.Errorf("app: audio dir: %w", err)
	}
	path := filepath.Join(s.audioDir, filename)
	if err := os.WriteFile(path, res.Audio, 0o644); err != nil {
		return "", fmt.Errorf("app: write audio: %w", err)
	}
	url := s.audioBase + filename
	if err := s.stories.UpdatePageAudio(ctx, storyID, seq, url); err != nil {
		return "", fmt.Errorf("app: persist audio url: %w", err)
	}
	return url, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
