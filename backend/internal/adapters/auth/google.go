package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ridopark/solo-adeventure/backend/internal/domain"
)

type Google struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthEndpoint string
	TokenEndpoint string
	UserEndpoint  string
	Client       *http.Client
}

func NewGoogle(clientID, clientSecret, redirectURI string, client *http.Client) *Google {
	if client == nil {
		client = http.DefaultClient
	}
	return &Google{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
		AuthEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint: "https://oauth2.googleapis.com/token",
		UserEndpoint:  "https://openidconnect.googleapis.com/v1/userinfo",
		Client:       client,
	}
}

func (g *Google) AuthURL(state string) string {
	q := url.Values{}
	q.Set("client_id", g.ClientID)
	q.Set("redirect_uri", g.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "openid email profile")
	q.Set("state", state)
	q.Set("access_type", "online")
	q.Set("prompt", "select_account")
	return g.AuthEndpoint + "?" + q.Encode()
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type userInfoResponse struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (g *Google) Exchange(ctx context.Context, code string) (domain.OAuthUserInfo, error) {
	form := url.Values{}
	form.Set("client_id", g.ClientID)
	form.Set("client_secret", g.ClientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", g.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: build token req: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: token request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: token status %d", resp.StatusCode)
	}
	var tok tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: token decode: %w", err)
	}

	ureq, err := http.NewRequestWithContext(ctx, http.MethodGet, g.UserEndpoint, nil)
	if err != nil {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: build user req: %w", err)
	}
	ureq.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	ureq.Header.Set("Accept", "application/json")

	uresp, err := g.Client.Do(ureq)
	if err != nil {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: userinfo request: %w", err)
	}
	defer uresp.Body.Close()
	if uresp.StatusCode < 200 || uresp.StatusCode >= 300 {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: userinfo status %d", uresp.StatusCode)
	}
	var ui userInfoResponse
	if err := json.NewDecoder(uresp.Body).Decode(&ui); err != nil {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: userinfo decode: %w", err)
	}
	if ui.Sub == "" {
		return domain.OAuthUserInfo{}, fmt.Errorf("auth.google: empty sub")
	}
	return domain.OAuthUserInfo{
		Sub:       ui.Sub,
		Email:     ui.Email,
		Name:      ui.Name,
		AvatarURL: ui.Picture,
	}, nil
}
