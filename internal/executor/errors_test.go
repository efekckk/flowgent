package executor

import (
	"errors"
	"testing"
)

func TestIsRetryable_classifies(t *testing.T) {
	cases := map[error]bool{
		ErrRateLimited:  true,
		ErrTransient5xx: true,
		ErrAuthFailed:   false,
		ErrValidation:   false,
		errors.New("random"): false,
	}
	for err, want := range cases {
		if got := IsRetryable(err); got != want {
			t.Errorf("IsRetryable(%v) = %v, want %v", err, got, want)
		}
	}
}

func TestIsRetryable_unwrapsWrappedError(t *testing.T) {
	wrapped := errors.New("slack 429: " + ErrRateLimited.Error())
	if IsRetryable(wrapped) {
		t.Errorf("plain string equality must not count as retryable")
	}
	properlyWrapped := errors.Join(ErrRateLimited, errors.New("slack 429"))
	if !IsRetryable(properlyWrapped) {
		t.Errorf("errors.Join with sentinel must be retryable")
	}
}
