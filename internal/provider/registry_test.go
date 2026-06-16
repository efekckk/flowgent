package provider

import (
	"encoding/json"
	"testing"
)

func TestRegistry_resolveProviderFromEnv(t *testing.T) {
	t.Setenv("FLOWGENT_OPENAI_KEY", "sk-test")
	r := NewRegistry()
	prov, err := r.For("openai")
	if err != nil {
		t.Fatalf("for: %v", err)
	}
	if _, ok := prov.(*OpenAI); !ok {
		t.Errorf("got %T", prov)
	}
}

func TestRegistry_anthropicFromEnv(t *testing.T) {
	t.Setenv("FLOWGENT_ANTHROPIC_KEY", "sk-ant-test")
	r := NewRegistry()
	prov, err := r.For("anthropic")
	if err != nil {
		t.Fatalf("for: %v", err)
	}
	if _, ok := prov.(*Anthropic); !ok {
		t.Errorf("got %T", prov)
	}
}

func TestRegistry_unknownProvider(t *testing.T) {
	r := NewRegistry()
	_, err := r.For("xyz")
	if err == nil {
		t.Fatalf("expected error for unknown provider")
	}
}

func TestRegistry_missingKeyIsError(t *testing.T) {
	t.Setenv("FLOWGENT_OPENAI_KEY", "")
	r := NewRegistry()
	_, err := r.For("openai")
	if err == nil {
		t.Fatalf("expected error: missing key")
	}
}

func TestRegistry_ForCredential_openai(t *testing.T) {
	r := NewRegistry()
	secret, _ := json.Marshal(map[string]any{"api_key": "sk-test"})
	p, err := r.ForCredential("openai", secret)
	if err != nil {
		t.Fatalf("for: %v", err)
	}
	if _, ok := p.(*OpenAI); !ok {
		t.Errorf("got %T", p)
	}
}

func TestRegistry_ForCredential_anthropic(t *testing.T) {
	r := NewRegistry()
	secret, _ := json.Marshal(map[string]any{"api_key": "sk-ant-test"})
	p, err := r.ForCredential("anthropic", secret)
	if err != nil {
		t.Fatalf("for: %v", err)
	}
	if _, ok := p.(*Anthropic); !ok {
		t.Errorf("got %T", p)
	}
}

func TestRegistry_ForCredential_missingApiKey(t *testing.T) {
	r := NewRegistry()
	secret, _ := json.Marshal(map[string]any{})
	_, err := r.ForCredential("openai", secret)
	if err == nil {
		t.Fatalf("expected error for missing api_key")
	}
}

func TestRegistry_ForCredential_unknownType(t *testing.T) {
	r := NewRegistry()
	_, err := r.ForCredential("xyz", []byte(`{}`))
	if err == nil {
		t.Fatalf("expected error for unknown type")
	}
}
