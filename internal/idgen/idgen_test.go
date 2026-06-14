package idgen

import (
	"strings"
	"testing"
	"time"
)

func TestNewID_prefixesCorrectly(t *testing.T) {
	cases := map[string]func() string{
		"usr_":   func() string { return NewUser() },
		"ws_":    func() string { return NewWorkspace() },
		"sess_":  func() string { return NewSession() },
		"cred_":  func() string { return NewCredential() },
		"wf_":    func() string { return NewWorkflow() },
		"wfv_":   func() string { return NewWorkflowVersion() },
		"run_":   func() string { return NewRun() },
		"nrun_":  func() string { return NewNodeRun() },
		"thr_":   func() string { return NewChatThread() },
		"msg_":   func() string { return NewChatMessage() },
	}
	for prefix, fn := range cases {
		got := fn()
		if !strings.HasPrefix(got, prefix) {
			t.Errorf("expected prefix %q, got %q", prefix, got)
		}
		if len(got) != len(prefix)+26 {
			t.Errorf("expected total length %d for %q, got %d", len(prefix)+26, got, len(got))
		}
	}
}

func TestNewID_sortableByTime(t *testing.T) {
	a := NewUser()
	time.Sleep(2 * time.Millisecond)
	b := NewUser()
	if a >= b {
		t.Fatalf("expected %q < %q (time-sorted)", a, b)
	}
}

func TestNewID_uniqueOver1k(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		id := NewUser()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id at i=%d: %s", i, id)
		}
		seen[id] = struct{}{}
	}
}
