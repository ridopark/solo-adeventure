package domain

import "errors"

var (
	ErrStoryNotFound  = errors.New("story not found")
	ErrInvalidChoice  = errors.New("invalid choice index")
	ErrStoryEnded     = errors.New("story has ended")
)
