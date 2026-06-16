package executor_test

import (
	"context"
	"errors"
	"testing"

	"github.com/efekckk/flowgent/internal/executor"
	"github.com/efekckk/flowgent/internal/registry"
)

type fakeCredentialResolver struct {
	calls []string
}

func (f *fakeCredentialResolver) Resolve(_ context.Context, credentialRef string) (map[string]any, error) {
	f.calls = append(f.calls, credentialRef)
	if credentialRef == "cred_unknown" {
		return nil, errors.New("not found")
	}
	return map[string]any{"id": credentialRef, "type": "openai"}, nil
}

func TestEngine_injectsCredentialIntoNodeInput(t *testing.T) {
	resolver := &fakeCredentialResolver{}
	reg := registry.New()
	rec := &recordingExec{out: map[string]any{"ok": true}}
	reg.Register("needs.credential", rec)

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n", Tool: "needs.credential", Credential: "cred_xyz", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg, executor.WithCredentialResolver(resolver))
	_, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("calls: %d", len(rec.calls))
	}
	got := rec.calls[0]
	cred, ok := got["__credential"].(map[string]any)
	if !ok {
		t.Fatalf("missing __credential in input: %+v", got)
	}
	if cred["id"] != "cred_xyz" {
		t.Errorf("credential id: %+v", cred)
	}
}

func TestEngine_missingCredentialResolverIsNoop(t *testing.T) {
	reg := registry.New()
	rec := &recordingExec{out: map[string]any{}}
	reg.Register("simple.tool", rec)

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n", Tool: "simple.tool", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg)
	_, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Errorf("calls: %d", len(rec.calls))
	}
}

func TestEngine_credentialResolutionFailureFailsRun(t *testing.T) {
	resolver := &fakeCredentialResolver{}
	reg := registry.New()
	reg.Register("needs.cred", &recordingExec{out: map[string]any{}})

	wf := executor.Workflow{
		Nodes: []executor.Node{
			{ID: "n", Tool: "needs.cred", Credential: "cred_unknown", Params: map[string]any{}},
		},
	}
	eng := executor.NewEngine(reg, executor.WithCredentialResolver(resolver))
	_, err := eng.Run(context.Background(), &wf, executor.RunOptions{TriggerKind: "manual"})
	if err == nil {
		t.Fatalf("expected error")
	}
}
