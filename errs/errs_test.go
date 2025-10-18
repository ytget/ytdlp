package errs

import (
	"errors"
	"testing"
)

func TestErrorConstants(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrVideoUnavailable",
			err:      ErrVideoUnavailable,
			expected: "video unavailable",
		},
		{
			name:     "ErrPrivate",
			err:      ErrPrivate,
			expected: "video is private",
		},
		{
			name:     "ErrAgeRestricted",
			err:      ErrAgeRestricted,
			expected: "age restricted",
		},
		{
			name:     "ErrCipherFailed",
			err:      ErrCipherFailed,
			expected: "cipher failed",
		},
		{
			name:     "ErrGeoBlocked",
			err:      ErrGeoBlocked,
			expected: "geo blocked",
		},
		{
			name:     "ErrRateLimited",
			err:      ErrRateLimited,
			expected: "rate limited",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected error message '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	// Test that errors are of the correct type
	if !errors.Is(ErrVideoUnavailable, ErrVideoUnavailable) {
		t.Error("ErrVideoUnavailable should be equal to itself")
	}

	if !errors.Is(ErrPrivate, ErrPrivate) {
		t.Error("ErrPrivate should be equal to itself")
	}

	if !errors.Is(ErrAgeRestricted, ErrAgeRestricted) {
		t.Error("ErrAgeRestricted should be equal to itself")
	}

	if !errors.Is(ErrCipherFailed, ErrCipherFailed) {
		t.Error("ErrCipherFailed should be equal to itself")
	}

	if !errors.Is(ErrGeoBlocked, ErrGeoBlocked) {
		t.Error("ErrGeoBlocked should be equal to itself")
	}

	if !errors.Is(ErrRateLimited, ErrRateLimited) {
		t.Error("ErrRateLimited should be equal to itself")
	}
}

func TestErrorUniqueness(t *testing.T) {
	// Test that different errors are not equal
	errorList := []error{
		ErrVideoUnavailable,
		ErrPrivate,
		ErrAgeRestricted,
		ErrCipherFailed,
		ErrGeoBlocked,
		ErrRateLimited,
	}

	for i, err1 := range errorList {
		for j, err2 := range errorList {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("Error %d and %d should not be equal", i, j)
			}
		}
	}
}
