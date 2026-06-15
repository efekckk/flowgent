// Package executor defines the workflow run loop, retry policy, and the
// sentinel error classes that tools wrap when they need the engine to make
// a retry vs. abort decision.
package executor

import "errors"

var (
	// ErrRateLimited — provider returned 429 or equivalent. Engine retries
	// with backoff (respecting any Retry-After signal the tool surfaces in
	// its output before returning the error).
	ErrRateLimited = errors.New("rate_limited")

	// ErrTransient5xx — server-side 500..599. Engine retries with backoff.
	ErrTransient5xx = errors.New("transient_5xx")

	// ErrAuthFailed — credential invalid/expired. Engine does NOT retry;
	// the user must fix the credential.
	ErrAuthFailed = errors.New("auth_failed")

	// ErrValidation — input failed the tool's own validation. Engine does
	// NOT retry because re-running will fail again with the same input.
	ErrValidation = errors.New("validation")
)

func IsRetryable(err error) bool {
	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrTransient5xx)
}
