package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// ExecuteResult is what a tool's executor returns. Port names the output
// port that "fired" — for most tools "main", for core.if "true"/"false".
type ExecuteResult struct {
	Output map[string]any
	Port   string
}

// Executor is the runtime contract for every tool. Implementations live in
// tools/<slug>/executor.go and are registered with the Registry at startup
// via Register().
type Executor interface {
	Execute(ctx context.Context, input map[string]any) (ExecuteResult, error)
}

type entry struct {
	Manifest Manifest
	Exec     Executor
}

type Registry struct {
	mu      sync.RWMutex
	entries map[string]*entry
	execs   map[string]Executor
}

func New() *Registry {
	return &Registry{
		entries: make(map[string]*entry),
		execs:   make(map[string]Executor),
	}
}

// Register attaches a Go executor to a slug. Manifests loaded by LoadFromDir
// are matched against this set; an unregistered slug is a load-time error.
func (r *Registry) Register(slug string, exec Executor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.execs[slug] = exec
}

// LoadFromDir walks dir for "<slug>/manifest.json" files, validates each,
// and stores it alongside its previously-registered executor. Tools without
// an executor are rejected so deployment mistakes fail fast.
func (r *Registry) LoadFromDir(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*", "manifest.json"))
	if err != nil {
		return fmt.Errorf("registry: glob: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("registry: read %s: %w", path, err)
		}
		m, err := ParseManifest(raw)
		if err != nil {
			return fmt.Errorf("registry: %s: %w", path, err)
		}
		exec, ok := r.execs[m.Slug]
		if !ok {
			return fmt.Errorf("registry: manifest %q has no registered executor", m.Slug)
		}
		r.entries[m.Slug] = &entry{Manifest: m, Exec: exec}
	}
	return nil
}

func (r *Registry) Get(slug string) (Executor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[slug]
	if !ok {
		return nil, false
	}
	return e.Exec, true
}

func (r *Registry) Manifest(slug string) (Manifest, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[slug]
	if !ok {
		return Manifest{}, false
	}
	return e.Manifest, true
}

// List returns all loaded manifests in slug-sorted order. Used by the AI
// agent (M4) to render the available-tools palette.
func (r *Registry) List() []Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Manifest, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e.Manifest)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out
}
