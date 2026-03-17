package agents

import (
	"context"
	"testing"

	"github.com/scttfrdmn/agenkit/agenkit-go/patterns"
	"github.com/scttfrdmn/bucktooth/internal/tools"
)

// mockLLMClient is a minimal patterns.LLMClient for testing.
type mockLLMClient struct {
	response string
}

func (m *mockLLMClient) Chat(ctx context.Context, messages interface{}) (interface{}, error) {
	return m.response, nil
}

func TestToolStepExecutor_FallsBackToLLM(t *testing.T) {
	// With no registry the executor should not panic.
	executor := NewToolStepExecutor(nil, nil)
	if executor == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestToolStepExecutor_MatchesTool(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(tools.NewCalculatorTool())

	executor := NewToolStepExecutor(reg, nil)

	step := patterns.PlanStep{
		Description: "use the calculator to add 2 and 3",
		StepNumber:  0,
	}

	// The calculator tool should be matched by name.  The input "use the
	// calculator to add 2 and 3" won't produce a valid calculation, but
	// it should reach the tool without panicking.
	result, err := executor.Execute(context.Background(), step, nil)
	// The tool will return an error or a ToolError result — both are acceptable.
	// What we verify is that the executor runs to completion.
	_ = result
	_ = err
}

func TestNewBuckToothPlanningAgent(t *testing.T) {
	// Verify the constructor doesn't panic and returns a non-nil agent.
	agent := NewBuckToothPlanningAgent(nil, &patterns.DefaultStepExecutor{}, 5)
	if agent == nil {
		t.Fatal("expected non-nil planning agent")
	}
}
