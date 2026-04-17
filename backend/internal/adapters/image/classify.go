package image

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// ErrTransient marks an image-provider error as safe to retry or fall back.
var ErrTransient = errors.New("image: transient")

// IsTransient reports whether err is a rate-limit, 5xx, timeout, or network error
// that the fallback composite should treat as retryable.
func IsTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrTransient) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var nerr net.Error
	if errors.As(err, &nerr) && nerr.Timeout() {
		return true
	}
	return false
}

// StatusToError converts an HTTP status into a transient-or-permanent error.
// Used by concrete provider adapters to normalize their error signal.
func StatusToError(status int) error {
	switch {
	case status == http.StatusTooManyRequests, status >= 500:
		return ErrTransient
	case status >= 400:
		return errors.New("image: permanent client error")
	}
	return nil
}
