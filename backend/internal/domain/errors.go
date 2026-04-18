package domain

import "errors"

var (
	ErrStoryNotFound  = errors.New("story not found")
	ErrInvalidChoice  = errors.New("invalid choice index")
	ErrStoryEnded     = errors.New("story has ended")
	ErrUnauthorized   = errors.New("authentication required")
	ErrForbidden      = errors.New("not the story owner")
	ErrSessionInvalid = errors.New("session invalid or expired")
	ErrPageNotFound   = errors.New("page not found")
	ErrTTSUnavailable   = errors.New("tts unavailable")
	ErrDepthUnavailable = errors.New("depth unavailable")
)
