// Package expression evaluates n8n-style "{{ ... }}" template expressions
// against a run-scoped context. Backed by expr-lang/expr — we map the
// dollar-prefixed identifiers ($trigger, $nodes, $json, $now, $env) onto
// regular Go variables before compiling.
package expression

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/expr-lang/expr"
)

type EvalContext struct {
	Trigger map[string]any
	Nodes   map[string]map[string]any
	Self    map[string]any
	Now     time.Time
	Env     map[string]string
}

// identRe matches Go-style identifiers (map keys accessed via dot notation).
var (
	triggerFieldRe = regexp.MustCompile(`\btrigger\.([a-zA-Z_][a-zA-Z0-9_]*)`)
	nodesFieldRe   = regexp.MustCompile(`\bnodes\.([a-zA-Z_][a-zA-Z0-9_]*)`)
	stringLitRe    = regexp.MustCompile(`"[^"]*"|'[^']*'`)
)

var dollarReplacer = strings.NewReplacer(
	"$trigger", "trigger",
	"$nodes", "nodes",
	"$json", "json",
	"$now", "now",
	"$env", "env",
)

// Evaluate compiles and runs a single expression in the provided context.
// The expression should NOT include the surrounding "{{ }}" markers — those
// are stripped by the template renderer before reaching here.
func Evaluate(expression string, ctx EvalContext) (any, error) {
	code := dollarReplacer.Replace(strings.TrimSpace(expression))

	if err := validateKeys(code, ctx); err != nil {
		return nil, fmt.Errorf("expression: %w", err)
	}

	env := buildEnv(ctx)
	program, err := expr.Compile(code, expr.Env(env))
	if err != nil {
		return nil, fmt.Errorf("expression: compile: %w", err)
	}
	out, err := expr.Run(program, env)
	if err != nil {
		return nil, fmt.Errorf("expression: run: %w", err)
	}
	return out, nil
}

// validateKeys checks that dot-accessed fields on trigger and nodes maps
// reference keys that actually exist, returning an error for missing keys.
func validateKeys(code string, ctx EvalContext) error {
	stripped := stringLitRe.ReplaceAllString(code, `""`)

	for _, m := range triggerFieldRe.FindAllStringSubmatch(stripped, -1) {
		key := m[1]
		if ctx.Trigger == nil {
			return fmt.Errorf("key %q not found in trigger (trigger is nil)", key)
		}
		if _, ok := ctx.Trigger[key]; !ok {
			return fmt.Errorf("key %q not found in trigger", key)
		}
	}

	for _, m := range nodesFieldRe.FindAllStringSubmatch(stripped, -1) {
		nodeID := m[1]
		if ctx.Nodes == nil {
			return fmt.Errorf("node %q not found in nodes (nodes is nil)", nodeID)
		}
		if _, ok := ctx.Nodes[nodeID]; !ok {
			return fmt.Errorf("node %q not found in nodes", nodeID)
		}
	}

	return nil
}

func buildEnv(ctx EvalContext) map[string]any {
	now := ctx.Now
	if now.IsZero() {
		now = time.Now()
	}
	return map[string]any{
		"trigger": ctx.Trigger,
		"nodes":   ctx.Nodes,
		"json":    ctx.Self,
		"now":     now,
		"env":     ctx.Env,
	}
}
