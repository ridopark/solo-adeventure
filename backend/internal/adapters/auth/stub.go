package auth

import (
	"context"
	"fmt"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

// Stub is a deterministic OAuthProvider for local dev without a real Google client.
// AuthURL returns a local /auth/google/dev-login handler URL; Exchange returns a
// fixed user per code. Enable with STORY_PROVIDER / OAUTH_PROVIDER=stub.
type Stub struct {
	BaseURL string
}

func NewStub(baseURL string) *Stub { return &Stub{BaseURL: baseURL} }

func (s *Stub) AuthURL(state string) string {
	return fmt.Sprintf("%s/auth/google/dev-login?state=%s", s.BaseURL, state)
}

func (s *Stub) Exchange(_ context.Context, code string) (domain.OAuthUserInfo, error) {
	return domain.OAuthUserInfo{
		Sub:       "dev-" + code,
		Email:     fmt.Sprintf("dev+%s@localhost", code),
		Name:      "Dev User",
		AvatarURL: "",
	}, nil
}
