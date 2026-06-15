package provider

import (
	"context"
	"errors"
	"sync"
)

// Mock is a deterministic ChatProvider for unit tests. Use Reply() to push
// scripted responses; Chat() pops them in order. Calls() returns the
// captured requests for assertions.
type Mock struct {
	mu       sync.Mutex
	replies  []ChatResponse
	calls    []ChatRequest
	errReply error
}

func NewMock() *Mock { return &Mock{} }

func (m *Mock) Reply(resp ChatResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.replies = append(m.replies, resp)
}

func (m *Mock) ReplyError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errReply = err
}

func (m *Mock) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, req)
	if m.errReply != nil {
		err := m.errReply
		m.errReply = nil
		return ChatResponse{}, err
	}
	if len(m.replies) == 0 {
		return ChatResponse{}, errors.New("mock: no replies scripted")
	}
	resp := m.replies[0]
	m.replies = m.replies[1:]
	return resp, nil
}

func (m *Mock) Calls() []ChatRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ChatRequest(nil), m.calls...)
}
