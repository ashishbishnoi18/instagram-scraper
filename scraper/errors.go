package scraper

import "errors"

var (
	ErrInvalidURL      = errors.New("invalid URL or input")
	ErrNotFound        = errors.New("resource not found")
	ErrPrivateResource = errors.New("private or restricted resource")
	ErrRateLimited     = errors.New("rate limited")
	ErrBlocked         = errors.New("blocked by target")
	ErrUpstreamChanged = errors.New("upstream schema changed")
	ErrContextCanceled = errors.New("context canceled")
)
