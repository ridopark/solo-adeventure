package store

import (
	"context"
	"sync"
	"time"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

type Memory struct {
	mu      sync.RWMutex
	stories map[string]domain.Story
	now     func() time.Time
}

func NewMemory() *Memory {
	return &Memory{
		stories: make(map[string]domain.Story),
		now:     func() time.Time { return time.Now().UTC() },
	}
}

func (m *Memory) Create(_ context.Context, s domain.Story) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stories[s.ID] = s
	return nil
}

func (m *Memory) Get(_ context.Context, id string) (domain.Story, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.stories[id]
	if !ok {
		return domain.Story{}, domain.ErrStoryNotFound
	}
	return s, nil
}

func (m *Memory) AppendPage(_ context.Context, storyID string, p domain.Page) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.stories[storyID]
	if !ok {
		return domain.ErrStoryNotFound
	}
	s.Pages = append(s.Pages, p)
	s.UpdatedAt = m.now()
	m.stories[storyID] = s
	return nil
}

func (m *Memory) ListByUser(_ context.Context, userID string, limit int) ([]domain.Story, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Story, 0)
	for _, s := range m.stories {
		if s.UserID == userID {
			out = append(out, s)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (m *Memory) UpdatePageAudio(_ context.Context, storyID string, idx int, audioURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.stories[storyID]
	if !ok {
		return domain.ErrStoryNotFound
	}
	if idx < 0 || idx >= len(s.Pages) {
		return domain.ErrStoryNotFound
	}
	s.Pages[idx].AudioURL = audioURL
	m.stories[storyID] = s
	return nil
}

func (m *Memory) AttachUser(_ context.Context, storyID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.stories[storyID]
	if !ok {
		return domain.ErrStoryNotFound
	}
	if s.UserID != "" {
		return domain.ErrForbidden
	}
	s.UserID = userID
	s.UpdatedAt = m.now()
	m.stories[storyID] = s
	return nil
}
