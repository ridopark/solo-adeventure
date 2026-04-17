package domain

import "time"

type User struct {
	ID        string    `json:"id"`
	GoogleSub string    `json:"-"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatarUrl,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type Session struct {
	ID        string    `json:"-"`
	UserID    string    `json:"-"`
	CreatedAt time.Time `json:"-"`
	ExpiresAt time.Time `json:"-"`
}

type OAuthUserInfo struct {
	Sub       string
	Email     string
	Name      string
	AvatarURL string
}
