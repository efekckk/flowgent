package executor

import (
	"sync"
	"time"
)

// NodeRecord captures one attempt (or one loop iteration) of a node's run.
// The handler layer persists these into node_runs rows.
type NodeRecord struct {
	Iteration  int
	Status     string
	Input      map[string]any
	Output     map[string]any
	Port       string
	Err        string
	Attempts   int
	StartedAt  time.Time
	FinishedAt time.Time
}

// RunState is the concurrency-safe state machine for a single workflow run.
// All methods are safe for parallel goroutines.
type RunState struct {
	mu      sync.RWMutex
	status  string
	error   string
	nodes   map[string]string
	history map[string][]NodeRecord
	latest  map[string]map[string]any
	ports   map[string]string
	inputs  map[string]map[string]any
}

func NewRunState(_ int) *RunState {
	return &RunState{
		status:  "running",
		nodes:   make(map[string]string),
		history: make(map[string][]NodeRecord),
		latest:  make(map[string]map[string]any),
		ports:   make(map[string]string),
		inputs:  make(map[string]map[string]any),
	}
}

func (s *RunState) SetRunStatus(status, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
	s.error = errMsg
}

func (s *RunState) RunStatus() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status, s.error
}

func (s *RunState) SetStatus(nodeID, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[nodeID] = status
}

func (s *RunState) Status(nodeID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nodes[nodeID]
}

func (s *RunState) Statuses() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.nodes))
	for k, v := range s.nodes {
		out[k] = v
	}
	return out
}

func (s *RunState) SetInput(nodeID string, input map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputs[nodeID] = input
}

func (s *RunState) Input(nodeID string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inputs[nodeID]
}

func (s *RunState) RecordOutput(nodeID string, iteration int, output map[string]any, port string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history[nodeID] = append(s.history[nodeID], NodeRecord{
		Iteration: iteration, Status: "succeeded", Output: output, Port: port,
	})
	s.latest[nodeID] = output
	s.ports[nodeID] = port
	s.nodes[nodeID] = "succeeded"
}

func (s *RunState) RecordFailure(nodeID string, iteration int, attempts int, errMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history[nodeID] = append(s.history[nodeID], NodeRecord{
		Iteration: iteration, Status: "failed", Err: errMsg, Attempts: attempts,
	})
	s.nodes[nodeID] = "failed"
}

func (s *RunState) LatestOutput(nodeID string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest[nodeID]
}

func (s *RunState) LatestPort(nodeID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ports[nodeID]
}

func (s *RunState) History(nodeID string) []NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]NodeRecord(nil), s.history[nodeID]...)
}

func (s *RunState) LatestOutputsMap() map[string]map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]map[string]any, len(s.latest))
	for k, v := range s.latest {
		out[k] = v
	}
	return out
}
