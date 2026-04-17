package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

type userCtxKey struct{}

// WithUser stores userID in request context; read with CurrentUserID.
func WithUser(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}
	return context.WithValue(ctx, userCtxKey{}, userID)
}

func CurrentUserID(ctx context.Context) string {
	if v, ok := ctx.Value(userCtxKey{}).(string); ok {
		return v
	}
	return ""
}

const SessionTTL = 30 * 24 * time.Hour

// BeginLogin returns the Google authorize URL to redirect the browser to.
// The state token is stored server-side via the caller (HTTP handler stores it
// in a short-lived cookie or map) -- this method is stateless.
func (s *Service) BeginLogin() (authURL, state string, err error) {
	if s.oauth == nil {
		return "", "", fmt.Errorf("app: oauth provider not configured")
	}
	state = randomToken(24)
	return s.oauth.AuthURL(state), state, nil
}

// CompleteLogin exchanges an OAuth code for user info, upserts the user, and
// creates a session. Returns the session id (to be set as the cookie value).
func (s *Service) CompleteLogin(ctx context.Context, code string) (domain.Session, domain.User, error) {
	if s.oauth == nil || s.users == nil || s.sessions == nil {
		return domain.Session{}, domain.User{}, fmt.Errorf("app: auth not configured")
	}
	info, err := s.oauth.Exchange(ctx, code)
	if err != nil {
		return domain.Session{}, domain.User{}, fmt.Errorf("app: exchange: %w", err)
	}
	user, err := s.users.GetByGoogleSub(ctx, info.Sub)
	if err != nil {
		user = domain.User{
			ID:        s.newID(),
			GoogleSub: info.Sub,
			Email:     info.Email,
			Name:      info.Name,
			AvatarURL: info.AvatarURL,
			CreatedAt: s.now(),
		}
		if err := s.users.Create(ctx, user); err != nil {
			return domain.Session{}, domain.User{}, fmt.Errorf("app: user create: %w", err)
		}
	}
	sess := domain.Session{
		ID:        randomToken(32),
		UserID:    user.ID,
		CreatedAt: s.now(),
		ExpiresAt: s.now().Add(SessionTTL),
	}
	if err := s.sessions.Create(ctx, sess); err != nil {
		return domain.Session{}, domain.User{}, fmt.Errorf("app: session create: %w", err)
	}
	return sess, user, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if s.sessions == nil || sessionID == "" {
		return nil
	}
	return s.sessions.Delete(ctx, sessionID)
}

func (s *Service) UserBySession(ctx context.Context, sessionID string) (domain.User, error) {
	if s.sessions == nil || s.users == nil {
		return domain.User{}, domain.ErrUnauthorized
	}
	sess, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return domain.User{}, err
	}
	return s.users.Get(ctx, sess.UserID)
}

func (s *Service) MyStories(ctx context.Context) ([]domain.Story, error) {
	uid := CurrentUserID(ctx)
	if uid == "" {
		return nil, domain.ErrUnauthorized
	}
	return s.stories.ListByUser(ctx, uid, 50)
}

func (s *Service) ClaimStory(ctx context.Context, storyID string) error {
	uid := CurrentUserID(ctx)
	if uid == "" {
		return domain.ErrUnauthorized
	}
	return s.stories.AttachUser(ctx, storyID, uid)
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
