package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/efekckk/flowgent/internal/provider"
)

type Deps struct {
	Provider   provider.ChatProvider
	KnownTools map[string]struct{}
	MaxRetries int
}

type RunRequest struct {
	Model           string
	UserMessage     string
	History         []provider.Message
	CurrentWorkflow json.RawMessage
}

type RunResult struct {
	AssistantText    string
	ToolName         string
	ProposedWorkflow json.RawMessage
	Patches          json.RawMessage
	ValidationErrors []string
	Usage            provider.TokenUsage
}

type Agent struct {
	deps Deps
}

func New(deps Deps) *Agent {
	if deps.MaxRetries <= 0 {
		deps.MaxRetries = 3
	}
	return &Agent{deps: deps}
}

// Run executes one turn of the conversation. It calls the provider, possibly
// dispatches a meta-tool, and retries up to MaxRetries times if the proposed
// workflow fails validation.
func (a *Agent) Run(ctx context.Context, req RunRequest) (RunResult, error) {
	messages := make([]provider.Message, 0, len(req.History)+2)
	messages = append(messages, provider.Message{
		Role:    "system",
		Content: a.buildSystemPrompt(req),
	})
	messages = append(messages, req.History...)
	messages = append(messages, provider.Message{Role: "user", Content: req.UserMessage})

	for attempt := 0; attempt < a.deps.MaxRetries; attempt++ {
		resp, err := a.deps.Provider.Chat(ctx, provider.ChatRequest{
			Model:    req.Model,
			Messages: messages,
			Tools:    MetaTools(),
		})
		if err != nil {
			return RunResult{}, fmt.Errorf("agent: provider call: %w", err)
		}

		// No tool call → plain text reply
		if len(resp.ToolCalls) == 0 {
			return RunResult{
				AssistantText: resp.Content,
				Usage:         resp.Usage,
			}, nil
		}

		tc := resp.ToolCalls[0]
		switch tc.Name {
		case "propose_workflow":
			errs := ValidateWorkflow(tc.Arguments, a.deps.KnownTools)
			if len(errs) == 0 {
				return RunResult{
					AssistantText:    resp.Content,
					ToolName:         "propose_workflow",
					ProposedWorkflow: tc.Arguments,
					Usage:            resp.Usage,
				}, nil
			}
			// Feed errors back and let the model retry
			messages = append(messages,
				provider.Message{
					Role:      "assistant",
					Content:   resp.Content,
					ToolCalls: []provider.ToolCall{tc},
				},
				provider.Message{
					Role:        "tool",
					ToolCallID:  tc.ID,
					Name:        tc.Name,
					ToolResults: []byte("Validation failed:\n" + FormatErrors(errs)),
				},
			)
		case "edit_workflow":
			return RunResult{
				AssistantText: resp.Content,
				ToolName:      "edit_workflow",
				Patches:       tc.Arguments,
				Usage:         resp.Usage,
			}, nil
		default:
			return RunResult{}, fmt.Errorf("agent: unknown meta-tool %q", tc.Name)
		}
	}
	return RunResult{}, errors.New("agent: exhausted retries after repeated validation failures")
}

func (a *Agent) buildSystemPrompt(req RunRequest) string {
	tools := make([]string, 0, len(a.deps.KnownTools))
	for t := range a.deps.KnownTools {
		tools = append(tools, t)
	}
	prompt := `You are Flowgent, a workflow builder. The user describes what they want; you respond by calling propose_workflow (for new workflows) or edit_workflow (to patch an existing one). Otherwise reply in plain text.

Available workflow tool slugs:
`
	for _, t := range tools {
		prompt += "- " + t + "\n"
	}
	if len(req.CurrentWorkflow) > 0 {
		prompt += "\nThe user is currently editing this workflow:\n" + string(req.CurrentWorkflow) + "\n"
	}
	return prompt
}
