package tools

import (
	"context"
	"testing"
)

func TestCalculatorTool_Execute(t *testing.T) {
	calc := NewCalculatorTool()
	ctx := context.Background()

	tests := []struct {
		name       string
		params     map[string]any
		wantResult float64
		wantErr    bool
		wantFail   bool
	}{
		{
			name:       "add",
			params:     map[string]any{"operation": "add", "a": float64(3), "b": float64(4)},
			wantResult: 7,
		},
		{
			name:       "subtract",
			params:     map[string]any{"operation": "subtract", "a": float64(10), "b": float64(3)},
			wantResult: 7,
		},
		{
			name:       "multiply",
			params:     map[string]any{"operation": "multiply", "a": float64(6), "b": float64(7)},
			wantResult: 42,
		},
		{
			name:       "divide",
			params:     map[string]any{"operation": "divide", "a": float64(15), "b": float64(3)},
			wantResult: 5,
		},
		{
			name:     "divide by zero",
			params:   map[string]any{"operation": "divide", "a": float64(1), "b": float64(0)},
			wantFail: true,
		},
		{
			name:     "modulo by zero",
			params:   map[string]any{"operation": "modulo", "a": float64(10), "b": float64(0)},
			wantFail: true,
		},
		{
			name:     "unknown operation",
			params:   map[string]any{"operation": "power", "a": float64(2), "b": float64(3)},
			wantFail: true,
		},
		{
			name:     "missing operation",
			params:   map[string]any{"a": float64(1), "b": float64(2)},
			wantFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.Execute(ctx, tt.params)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantFail {
				if result.Success {
					t.Errorf("expected failure, got success with data=%v", result.Data)
				}
				return
			}
			if !result.Success {
				t.Errorf("expected success, got error: %s", result.Error)
				return
			}
			got, ok := result.Data.(float64)
			if !ok {
				t.Fatalf("expected float64 result, got %T", result.Data)
			}
			if got != tt.wantResult {
				t.Errorf("got %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestCalculatorTool_JSONInput(t *testing.T) {
	calc := NewCalculatorTool()
	ctx := context.Background()

	// Simulate how ReActAgent passes parameters: {"input": "<JSON string>"}
	result, err := calc.Execute(ctx, map[string]any{
		"input": `{"operation":"multiply","a":42,"b":7}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Error)
	}
	got, ok := result.Data.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", result.Data)
	}
	if got != 294 {
		t.Errorf("got %v, want 294", got)
	}
}
