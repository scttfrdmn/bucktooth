package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scttfrdmn/agenkit/agenkit-go/adapter/llm"
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
	"github.com/scttfrdmn/agenkit/agenkit-go/patterns"
	"github.com/scttfrdmn/bucktooth/internal/tools"
)

// ToolStepExecutor executes plan steps by matching them to registered tools or falling
// back to a direct LLM completion for free-form steps.
type ToolStepExecutor struct {
	registry *tools.Registry
	llm      *llm.AnthropicLLM
}

// NewToolStepExecutor creates a new ToolStepExecutor.
func NewToolStepExecutor(registry *tools.Registry, llmClient *llm.AnthropicLLM) *ToolStepExecutor {
	return &ToolStepExecutor{
		registry:  registry,
		llm:       llmClient,
	}
}

// Execute implements patterns.StepExecutor.
//
// It first checks whether any registered tool name appears in the step description;
// if so it invokes that tool with the step description as input. Otherwise it calls
// the LLM to handle the step as a free-form completion.
func (e *ToolStepExecutor) Execute(ctx context.Context, step patterns.PlanStep, stepContext map[string]interface{}) (interface{}, error) {
	if e.registry != nil {
		for _, tool := range e.registry.GetAll() {
			if strings.Contains(strings.ToLower(step.Description), strings.ToLower(tool.Name())) {
				result, err := tool.Execute(ctx, map[string]interface{}{
					"input": step.Description,
				})
				if err != nil {
					return nil, fmt.Errorf("tool %s failed: %w", tool.Name(), err)
				}
				if result != nil && !result.Success {
					return nil, fmt.Errorf("tool %s error: %s", tool.Name(), result.Error)
				}
				if result != nil {
					return result.Data, nil
				}
				return nil, nil
			}
		}
	}

	// Fallback: ask the LLM to handle the step.
	contextSummary := buildContextSummary(stepContext)
	prompt := step.Description
	if contextSummary != "" {
		prompt = fmt.Sprintf("%s\n\nContext from previous steps:\n%s", step.Description, contextSummary)
	}

	response, err := e.llm.Complete(ctx, []*agenkit.Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM step execution failed: %w", err)
	}

	return response.ContentString(), nil
}

// buildContextSummary converts the step context map to a readable string.
func buildContextSummary(ctx map[string]interface{}) string {
	if len(ctx) == 0 {
		return ""
	}

	parts := make([]string, 0, len(ctx))
	for k, v := range ctx {
		switch val := v.(type) {
		case string:
			parts = append(parts, fmt.Sprintf("%s: %s", k, val))
		default:
			if b, err := json.Marshal(val); err == nil {
				parts = append(parts, fmt.Sprintf("%s: %s", k, string(b)))
			}
		}
	}

	return strings.Join(parts, "\n")
}

const buckToothPlanningSystemPrompt = `You are BuckTooth, a helpful AI assistant and planning agent.

When given a complex task, break it into clear, actionable steps.

Format your plan as:
Goal: [overall goal]
Steps:
1. [first step]
2. [second step]
...

Guidelines:
- Make steps concrete and actionable
- Use available tools where appropriate (calculator, web_search, calendar, filesystem)
- Keep steps focused and achievable
- Include a final step to summarize results`

// NewBuckToothPlanningAgent creates a PlanningAgent with the BuckTooth system prompt.
func NewBuckToothPlanningAgent(llmClient patterns.LLMClient, executor patterns.StepExecutor, maxSteps int) *patterns.PlanningAgent {
	if maxSteps <= 0 {
		maxSteps = 10
	}

	return patterns.NewPlanningAgent(llmClient, executor, &patterns.PlanningAgentConfig{
		MaxSteps:        maxSteps,
		AllowReplanning: true,
		SystemPrompt:    buckToothPlanningSystemPrompt,
	})
}
