package llmchat

import (
	"context"
	"errors"
	"testing"

	"github.com/efekckk/flowgent/internal/provider"
)

type fakeProvider struct {
	resp provider.ChatResponse
	err  error
}

func (f *fakeProvider) Chat(_ context.Context, _ provider.ChatRequest) (provider.ChatResponse, error) {
	if f.err != nil {
		return provider.ChatResponse{}, f.err
	}
	return f.resp, nil
}

type fakeResolver struct {
	prov provider.ChatProvider
	err  error
}

func (f *fakeResolver) ResolveForNodeCredential(_ context.Context, _ map[string]any) (provider.ChatProvider, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.prov, nil
}

func TestExecute_returnsContent(t *testing.T) {
	fp := &fakeProvider{resp: provider.ChatResponse{
		Content: "Hi there!",
		Model:   "gpt-4o",
		Usage:   provider.TokenUsage{TotalTokens: 42},
	}}
	e := New(&fakeResolver{prov: fp})
	res, err := e.Execute(context.Background(), map[string]any{
		"model": "gpt-4o",
		"user":  "Hi",
		"__credential": map[string]any{"type": "openai", "id": "cred_1"},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Output["content"] != "Hi there!" {
		t.Errorf("content: %+v", res.Output)
	}
	if res.Output["tokens"] != 42 {
		t.Errorf("tokens: %+v", res.Output["tokens"])
	}
}

func TestExecute_missingUserIsError(t *testing.T) {
	fp := &fakeProvider{resp: provider.ChatResponse{Content: "x"}}
	e := New(&fakeResolver{prov: fp})
	_, err := e.Execute(context.Background(), map[string]any{
		"model": "gpt-4o",
		"__credential": map[string]any{"type": "openai"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestExecute_missingCredentialIsError(t *testing.T) {
	fp := &fakeProvider{resp: provider.ChatResponse{Content: "x"}}
	e := New(&fakeResolver{prov: fp})
	_, err := e.Execute(context.Background(), map[string]any{
		"model": "gpt-4o", "user": "hi",
	})
	if err == nil {
		t.Fatalf("expected error: no credential")
	}
}

func TestExecute_providerErrorBubbles(t *testing.T) {
	e := New(&fakeResolver{prov: &fakeProvider{err: errors.New("boom")}})
	_, err := e.Execute(context.Background(), map[string]any{
		"model": "gpt-4o", "user": "hi",
		"__credential": map[string]any{"type": "openai"},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}
