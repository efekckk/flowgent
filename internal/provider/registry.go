package provider

import (
	"fmt"
	"os"
)

// Registry resolves provider slugs ("openai", "anthropic") to live
// ChatProvider instances. Keys come from env vars in M4; M6 will move
// this to encrypted credentials stored in the credentials table.
type Registry struct {
	openAIBaseURL    string
	anthropicBaseURL string
}

func NewRegistry() *Registry {
	return &Registry{
		openAIBaseURL:    "https://api.openai.com/v1",
		anthropicBaseURL: "https://api.anthropic.com/v1",
	}
}

// WithOpenAIBaseURL overrides the OpenAI endpoint (used by tests).
func (r *Registry) WithOpenAIBaseURL(u string) *Registry { r.openAIBaseURL = u; return r }

// WithAnthropicBaseURL overrides the Anthropic endpoint (used by tests).
func (r *Registry) WithAnthropicBaseURL(u string) *Registry { r.anthropicBaseURL = u; return r }

func (r *Registry) For(slug string) (ChatProvider, error) {
	switch slug {
	case "openai":
		key := os.Getenv("FLOWGENT_OPENAI_KEY")
		if key == "" {
			return nil, fmt.Errorf("provider: FLOWGENT_OPENAI_KEY env var is empty")
		}
		return NewOpenAI(r.openAIBaseURL, key), nil
	case "anthropic":
		key := os.Getenv("FLOWGENT_ANTHROPIC_KEY")
		if key == "" {
			return nil, fmt.Errorf("provider: FLOWGENT_ANTHROPIC_KEY env var is empty")
		}
		return NewAnthropic(r.anthropicBaseURL, key), nil
	default:
		return nil, fmt.Errorf("provider: unknown provider %q", slug)
	}
}
