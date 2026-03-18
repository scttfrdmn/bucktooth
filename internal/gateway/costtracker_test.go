package gateway

import (
	"testing"
)

func TestCalculateCostKnownModel(t *testing.T) {
	// claude-sonnet-4-5: $3/1M input, $15/1M output
	// 1M input + 1M output => $3 + $15 = $18
	cost := calculateCost("anthropic", "claude-sonnet-4-5", 1_000_000, 1_000_000)
	if cost != 18.0 {
		t.Fatalf("expected 18.0, got %f", cost)
	}
}

func TestCalculateCostPartialTokens(t *testing.T) {
	// gpt-4o: $5/1M input, $15/1M output
	// 500k input + 250k output => $2.50 + $3.75 = $6.25
	cost := calculateCost("openai", "gpt-4o", 500_000, 250_000)
	want := 6.25
	if cost != want {
		t.Fatalf("expected %.2f, got %.2f", want, cost)
	}
}

func TestCalculateCostUnknownProvider(t *testing.T) {
	cost := calculateCost("unknown-provider", "some-model", 1_000_000, 1_000_000)
	if cost != 0.0 {
		t.Fatalf("expected 0.0 for unknown provider, got %f", cost)
	}
}

func TestCalculateCostUnknownModel(t *testing.T) {
	cost := calculateCost("anthropic", "unknown-model-x", 1_000_000, 1_000_000)
	if cost != 0.0 {
		t.Fatalf("expected 0.0 for unknown model, got %f", cost)
	}
}

func TestCostTrackerTrackAndSummary(t *testing.T) {
	ct := NewCostTracker()

	// Track 500k input + 100k output for claude-sonnet-4-5
	// cost = 0.5 * 3 + 0.1 * 15 = 1.5 + 1.5 = 3.0 USD
	ct.Track("anthropic", "claude-sonnet-4-5", 500_000, 100_000)

	summary := ct.Summary()
	if summary.TotalTokensIn != 500_000 {
		t.Fatalf("expected 500000 tokens_in, got %d", summary.TotalTokensIn)
	}
	if summary.TotalTokensOut != 100_000 {
		t.Fatalf("expected 100000 tokens_out, got %d", summary.TotalTokensOut)
	}
	if summary.TotalCostUSD != 3.0 {
		t.Fatalf("expected 3.0 cost_usd, got %f", summary.TotalCostUSD)
	}
	if len(summary.ByModel) != 1 {
		t.Fatalf("expected 1 model entry, got %d", len(summary.ByModel))
	}
	if summary.ByModel[0].Model != "claude-sonnet-4-5" {
		t.Fatalf("unexpected model name: %s", summary.ByModel[0].Model)
	}
}

func TestCostTrackerMultipleModels(t *testing.T) {
	ct := NewCostTracker()

	ct.Track("anthropic", "claude-sonnet-4-5", 1_000_000, 0)  // $3.0
	ct.Track("openai", "gpt-4o", 1_000_000, 0)                // $5.0
	ct.Track("anthropic", "claude-sonnet-4-5", 1_000_000, 0)  // $3.0 more

	summary := ct.Summary()
	if summary.TotalCostUSD != 11.0 {
		t.Fatalf("expected total 11.0, got %f", summary.TotalCostUSD)
	}
	if len(summary.ByModel) != 2 {
		t.Fatalf("expected 2 model entries, got %d", len(summary.ByModel))
	}
}

func TestCostTrackerUnknownModelGraceful(t *testing.T) {
	ct := NewCostTracker()
	// Should not panic; cost should be 0
	ct.Track("mystery-provider", "mystery-model", 999, 999)

	summary := ct.Summary()
	if summary.TotalCostUSD != 0.0 {
		t.Fatalf("expected 0.0 for unknown model, got %f", summary.TotalCostUSD)
	}
	// Tokens are still recorded
	if summary.TotalTokensIn != 999 {
		t.Fatalf("expected 999 tokens_in, got %d", summary.TotalTokensIn)
	}
}

func TestCostTrackerRingBufferBound(t *testing.T) {
	ct := NewCostTracker()
	// Insert more than costRingSize entries; tracker should not OOM or panic.
	for i := 0; i < costRingSize+100; i++ {
		ct.Track("anthropic", "claude-haiku-4-5", 100, 100)
	}
	summary := ct.Summary()
	if summary.TotalTokensIn == 0 {
		t.Fatal("expected non-zero total tokens after ring buffer wrap")
	}
}

func TestCostTrackerSummaryEmpty(t *testing.T) {
	ct := NewCostTracker()
	summary := ct.Summary()
	if summary.TotalCostUSD != 0 {
		t.Fatalf("expected 0.0 on empty tracker, got %f", summary.TotalCostUSD)
	}
	if len(summary.ByModel) != 0 {
		t.Fatalf("expected empty ByModel slice, got %d entries", len(summary.ByModel))
	}
}
