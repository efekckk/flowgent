package auth

import (
	"strings"
	"sync"
	"time"
)

// LoginThrottle counts failed login attempts per email within a sliding
// window. It is intentionally in-memory: M1 is single-process, and for the
// 100-user demo this is plenty. Future milestones can move it to Postgres
// or Redis.
type LoginThrottle struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	now     func() time.Time
	buckets map[string][]time.Time
}

func NewLoginThrottle(max int, window time.Duration, now func() time.Time) *LoginThrottle {
	if now == nil {
		now = time.Now
	}
	return &LoginThrottle{
		max:     max,
		window:  window,
		now:     now,
		buckets: make(map[string][]time.Time),
	}
}

// AllowAndRecordFail returns true if a fresh failure is allowed without
// triggering a lockout. Returns false when the lockout threshold is hit;
// the caller should respond with 429.
func (t *LoginThrottle) AllowAndRecordFail(email string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(email))
	cutoff := t.now().Add(-t.window)
	bucket := t.buckets[key]
	pruned := bucket[:0]
	for _, ts := range bucket {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}
	if len(pruned) >= t.max {
		t.buckets[key] = pruned
		return false
	}
	pruned = append(pruned, t.now())
	t.buckets[key] = pruned
	return true
}

func (t *LoginThrottle) Reset(email string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.buckets, strings.ToLower(strings.TrimSpace(email)))
}
