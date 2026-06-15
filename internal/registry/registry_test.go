package registry_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/efekckk/flowgent/internal/registry"
)

type fakeExec struct{ slug string }

func (f *fakeExec) Execute(ctx context.Context, in map[string]any) (registry.ExecuteResult, error) {
	return registry.ExecuteResult{Output: map[string]any{"slug": f.slug}, Port: "main"}, nil
}

func TestRegistry_LoadFromDir(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "core.wait", `{
		"slug": "core.wait", "version": "1.0.0", "display_name": "Wait",
		"category": "control", "description": "..."
	}`)
	writeManifest(t, dir, "core.set", `{
		"slug": "core.set", "version": "1.0.0", "display_name": "Set",
		"category": "transform", "description": "..."
	}`)

	r := registry.New()
	r.Register("core.wait", &fakeExec{slug: "core.wait"})
	r.Register("core.set", &fakeExec{slug: "core.set"})
	if err := r.LoadFromDir(dir); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := len(r.List()); got != 2 {
		t.Errorf("want 2 tools, got %d", got)
	}
	if _, ok := r.Get("core.wait"); !ok {
		t.Errorf("core.wait missing")
	}
}

func TestRegistry_Get_unknownReturnsFalse(t *testing.T) {
	r := registry.New()
	if _, ok := r.Get("nope"); ok {
		t.Errorf("Get should return false on unknown slug")
	}
}

func TestRegistry_LoadFromDir_rejectsManifestWithoutExecutor(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "no.exec", `{
		"slug": "no.exec", "version": "1.0.0", "display_name": "X",
		"category": "y", "description": "z"
	}`)
	r := registry.New()
	err := r.LoadFromDir(dir)
	if err == nil {
		t.Fatalf("expected error: manifest without executor")
	}
}

func writeManifest(t *testing.T, dir, slug, body string) {
	t.Helper()
	sub := filepath.Join(dir, slug)
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
