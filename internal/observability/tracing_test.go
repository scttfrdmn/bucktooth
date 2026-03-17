package observability

import (
	"context"
	"testing"

	"github.com/scttfrdmn/bucktooth/internal/config"
)

func TestInitTracer_Disabled(t *testing.T) {
	cfg := config.TracingConfig{
		Enabled: false,
	}

	shutdown, err := InitTracer(cfg)
	if err != nil {
		t.Fatalf("unexpected error when tracing disabled: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown function")
	}

	// Calling shutdown should not error.
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}
