package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

func TestMemory_CreateAndGet(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	story := domain.Story{ID: "s-1", Topic: "haunted lighthouse"}

	require.NoError(t, m.Create(ctx, story))
	got, err := m.Get(ctx, "s-1")
	require.NoError(t, err)
	assert.Equal(t, "haunted lighthouse", got.Topic)
}

func TestMemory_Get_NotFound(t *testing.T) {
	m := NewMemory()
	_, err := m.Get(context.Background(), "missing")
	assert.True(t, errors.Is(err, domain.ErrStoryNotFound))
}

func TestMemory_AppendPage(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	require.NoError(t, m.Create(ctx, domain.Story{ID: "s-1"}))

	require.NoError(t, m.AppendPage(ctx, "s-1", domain.Page{Index: 0, Narrative: "first"}))
	require.NoError(t, m.AppendPage(ctx, "s-1", domain.Page{Index: 1, Narrative: "second"}))

	got, err := m.Get(ctx, "s-1")
	require.NoError(t, err)
	require.Len(t, got.Pages, 2)
	assert.Equal(t, "first", got.Pages[0].Narrative)
	assert.Equal(t, "second", got.Pages[1].Narrative)
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestMemory_AppendPage_MissingStory(t *testing.T) {
	m := NewMemory()
	err := m.AppendPage(context.Background(), "nope", domain.Page{})
	assert.True(t, errors.Is(err, domain.ErrStoryNotFound))
}

func TestMemory_ConcurrentAppends(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	require.NoError(t, m.Create(ctx, domain.Story{ID: "s-1"}))

	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			_ = m.AppendPage(ctx, "s-1", domain.Page{Index: i, Narrative: fmt.Sprintf("p%d", i)})
		}()
	}
	wg.Wait()

	got, err := m.Get(ctx, "s-1")
	require.NoError(t, err)
	assert.Len(t, got.Pages, n)
}
