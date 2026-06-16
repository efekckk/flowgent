// Package runlog provides an in-process fan-out so multiple SSE subscribers
// can tail the same run's logs. The engine's LogSink writes to both the
// database (via storage.RunLogRepo) AND publishes to the streamer; SSE
// handlers subscribe and unsubscribe. The streamer is purely in-memory —
// reconnecting clients catch up from storage via ?since=<last_id>.
package runlog

import (
	"context"
	"sync"
)

type Event struct {
	RunID   string `json:"run_id"`
	NodeID  string `json:"node_id"`
	Level   string `json:"level"`
	Message string `json:"message"`
	At      string `json:"at"` // RFC3339Nano
}

// Streamer fans out events keyed by run id. Publish never blocks: slow
// subscribers drop events rather than back-pressure the engine.
type Streamer struct {
	mu   sync.RWMutex
	subs map[string]map[chan Event]struct{}
}

func New() *Streamer {
	return &Streamer{subs: make(map[string]map[chan Event]struct{})}
}

func (s *Streamer) Publish(_ context.Context, e Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.subs[e.RunID] {
		select {
		case ch <- e:
		default:
			// slow subscriber; drop event rather than block the engine
		}
	}
}

// Subscribe returns a channel of events for runID and an unsubscribe func.
// The unsubscribe func closes the channel; calling it twice is safe via
// sync.Once.
func (s *Streamer) Subscribe(runID string) (<-chan Event, func()) {
	ch := make(chan Event, 32)
	s.mu.Lock()
	if s.subs[runID] == nil {
		s.subs[runID] = make(map[chan Event]struct{})
	}
	s.subs[runID][ch] = struct{}{}
	s.mu.Unlock()

	var once sync.Once
	return ch, func() {
		once.Do(func() {
			s.mu.Lock()
			if subs, ok := s.subs[runID]; ok {
				delete(subs, ch)
				if len(subs) == 0 {
					delete(s.subs, runID)
				}
			}
			s.mu.Unlock()
			close(ch)
		})
	}
}
