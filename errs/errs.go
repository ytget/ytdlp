package errs

import (
	"errors"
)

var (
	// ErrVideoUnavailable indicates that the requested video cannot be accessed.
	ErrVideoUnavailable = errors.New("video unavailable")
	// ErrPrivate indicates that the video is private and cannot be downloaded.
	ErrPrivate = errors.New("video is private")
	// ErrAgeRestricted indicates that the video has an age restriction.
	ErrAgeRestricted = errors.New("age restricted")
	// ErrCipherFailed indicates failure during signature deciphering.
	ErrCipherFailed = errors.New("cipher failed")
	// ErrGeoBlocked indicates the video is not available in the current region.
	ErrGeoBlocked = errors.New("geo blocked")
	// ErrRateLimited indicates throttling or rate limiting by the remote service.
	ErrRateLimited = errors.New("rate limited")
)
