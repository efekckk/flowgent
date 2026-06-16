// Package llmchat implements "llm.chat" — a workflow node that calls a chat
// provider with system + user messages. The engine resolves the node's
// `credential` reference, decrypts it, and injects the resulting credential
// payload into the input as __credential.
package llmchat

import (
	"context"
	"fmt"

	"github.com/efekckk/flowgent/internal/provider"
	"github.com/efekckk/flowgent/internal/registry"
)

// ProviderResolver is the contract the engine satisfies: given a node's
// resolved input (which includes __credential metadata), return a live
// ChatProvider to call.
type ProviderResolver interface {
	ResolveForNodeCredential(ctx context.Context, input map[string]any) (provider.ChatProvider, error)
}

type Executor struct {
	resolver ProviderResolver
}

func New(resolver ProviderResolver) *Executor { return &Executor{resolver: resolver} }

func (e *Executor) Execute(ctx context.Context, input map[string]any) (registry.ExecuteResult, error) {
	model, _ := input["model"].(string)
	if model == "" {
		return registry.ExecuteResult{}, fmt.Errorf("llm.chat: missing \"model\"")
	}
	userMsg, _ := input["user"].(string)
	if userMsg == "" {
		return registry.ExecuteResult{}, fmt.Errorf("llm.chat: missing \"user\"")
	}
	if _, hasCred := input["__credential"]; !hasCred {
		return registry.ExecuteResult{}, fmt.Errorf("llm.chat: node requires a credential reference")
	}
	prov, err := e.resolver.ResolveForNodeCredential(ctx, input)
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("llm.chat: resolve credential: %w", err)
	}

	messages := make([]provider.Message, 0, 2)
	if sys, _ := input["system"].(string); sys != "" {
		messages = append(messages, provider.Message{Role: "system", Content: sys})
	}
	messages = append(messages, provider.Message{Role: "user", Content: userMsg})

	resp, err := prov.Chat(ctx, provider.ChatRequest{
		Model:    model,
		Messages: messages,
	})
	if err != nil {
		return registry.ExecuteResult{}, fmt.Errorf("llm.chat: %w", err)
	}
	return registry.ExecuteResult{
		Output: map[string]any{
			"content": resp.Content,
			"model":   resp.Model,
			"tokens":  resp.Usage.TotalTokens,
		},
		Port: "main",
	}, nil
}
