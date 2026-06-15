package executor

import (
	"sync"
	"testing"
)

func TestRunState_concurrentWriteIsSafe(t *testing.T) {
	st := NewRunState(0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			st.SetStatus("n", "running")
			st.RecordOutput("n", 0, map[string]any{"i": i}, "main")
		}(i)
	}
	wg.Wait()
	if st.Status("n") != "succeeded" {
		// RecordOutput flips status to "succeeded"; the last writer wins
		t.Errorf("status: %s", st.Status("n"))
	}
}

func TestRunState_recordOutputPerIteration(t *testing.T) {
	st := NewRunState(0)
	st.RecordOutput("n", 0, map[string]any{"v": 1}, "main")
	st.RecordOutput("n", 1, map[string]any{"v": 2}, "main")

	latest := st.LatestOutput("n")
	if latest["v"] != 2 {
		t.Errorf("latest: %+v", latest)
	}
	history := st.History("n")
	if len(history) != 2 {
		t.Errorf("history: %d", len(history))
	}
}
