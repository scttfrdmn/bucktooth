package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// CalculatorTool performs basic arithmetic operations.
type CalculatorTool struct{}

// NewCalculatorTool creates a new calculator tool.
func NewCalculatorTool() *CalculatorTool {
	return &CalculatorTool{}
}

func (t *CalculatorTool) Name() string { return "calculator" }

func (t *CalculatorTool) Description() string {
	return `Performs arithmetic operations. Pass parameters as JSON: {"operation":"add|subtract|multiply|divide|modulo","a":<number>,"b":<number>}`
}

func (t *CalculatorTool) Execute(ctx context.Context, params map[string]any) (*agenkit.ToolResult, error) {
	// ReActAgent passes all tool input as {"input": "<string>"} — try to parse it as JSON.
	if raw, ok := params["input"].(string); ok {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			params = decoded
		}
	}

	operation, ok := params["operation"].(string)
	if !ok || operation == "" {
		return agenkit.NewToolError("missing required parameter: operation"), nil
	}

	a, err := toFloat(params["a"])
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("invalid parameter 'a': %v", err)), nil
	}
	b, err := toFloat(params["b"])
	if err != nil {
		return agenkit.NewToolError(fmt.Sprintf("invalid parameter 'b': %v", err)), nil
	}

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return agenkit.NewToolError("division by zero"), nil
		}
		result = a / b
	case "modulo":
		if b == 0 {
			return agenkit.NewToolError("modulo by zero"), nil
		}
		result = math.Mod(a, b)
	default:
		return agenkit.NewToolError(fmt.Sprintf("unknown operation: %s (must be add|subtract|multiply|divide|modulo)", operation)), nil
	}

	return agenkit.NewToolResult(result), nil
}

func toFloat(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case json.Number:
		return val.Float64()
	case string:
		return strconv.ParseFloat(val, 64)
	case nil:
		return 0, fmt.Errorf("value is nil")
	default:
		return strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
	}
}
