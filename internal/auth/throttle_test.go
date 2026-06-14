package auth

import (
	"testing"
	"time"
)

func TestThrottle_allowsUnderLimit(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	thr := NewLoginThrottle(5, 15*time.Minute, clock)
	for i := 0; i < 5; i++ {
		if !thr.AllowAndRecordFail("a@example.com") {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
}

func TestThrottle_blocksAtLimit(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	thr := NewLoginThrottle(5, 15*time.Minute, clock)
	for i := 0; i < 5; i++ {
		_ = thr.AllowAndRecordFail("b@example.com")
	}
	if thr.AllowAndRecordFail("b@example.com") {
		t.Fatalf("6th attempt should be blocked")
	}
}

func TestThrottle_recoversAfterWindow(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	thr := NewLoginThrottle(5, 15*time.Minute, clock)
	for i := 0; i < 5; i++ {
		_ = thr.AllowAndRecordFail("c@example.com")
	}
	now = now.Add(16 * time.Minute)
	if !thr.AllowAndRecordFail("c@example.com") {
		t.Fatalf("attempt after window must be allowed")
	}
}

func TestThrottle_Reset(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	thr := NewLoginThrottle(2, 15*time.Minute, clock)
	_ = thr.AllowAndRecordFail("d@example.com")
	_ = thr.AllowAndRecordFail("d@example.com")
	thr.Reset("d@example.com")
	if !thr.AllowAndRecordFail("d@example.com") {
		t.Fatalf("reset should clear counter")
	}
}
