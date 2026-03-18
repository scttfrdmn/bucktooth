package gateway

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var llmCostDollarsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "bucktooth_llm_cost_dollars_total",
	Help: "Total LLM cost in USD.",
}, []string{"provider", "model"})

// pricingTable holds static per-1M-token prices [input, output] in USD.
var pricingTable = map[string]map[string][2]float64{
	"anthropic": {
		"claude-opus-4":              {15.0, 75.0},
		"claude-sonnet-4-5":          {3.0, 15.0},
		"claude-sonnet-4-5-20250220": {3.0, 15.0},
		"claude-haiku-4-5":           {0.25, 1.25},
		"claude-haiku-4-5-20251001":  {0.25, 1.25},
	},
	"openai": {
		"gpt-4o":      {5.0, 15.0},
		"gpt-4o-mini": {0.15, 0.60},
	},
}

// CostEntry records a single LLM invocation.
type CostEntry struct {
	Provider  string
	Model     string
	TokensIn  uint64
	TokensOut uint64
	CostUSD   float64
}

// ModelCost holds cumulative per-model totals.
type ModelCost struct {
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	TokensIn  uint64  `json:"tokens_in"`
	TokensOut uint64  `json:"tokens_out"`
	CostUSD   float64 `json:"cost_usd"`
}

// CostSummary is the aggregate returned by CostTracker.Summary().
type CostSummary struct {
	TotalCostUSD   float64     `json:"total_cost_usd"`
	TotalTokensIn  uint64      `json:"total_tokens_in"`
	TotalTokensOut uint64      `json:"total_tokens_out"`
	ByModel        []ModelCost `json:"by_model"`
}

const costRingSize = 10000

// CostTracker records LLM token consumption and calculates USD cost.
// It uses a ring buffer to bound memory while maintaining cumulative totals.
type CostTracker struct {
	mu      sync.Mutex
	entries [costRingSize]CostEntry
	head    int
	size    int
	totals  map[string]*ModelCost // key: "provider/model"
}

// NewCostTracker creates a new CostTracker.
func NewCostTracker() *CostTracker {
	return &CostTracker{
		totals: make(map[string]*ModelCost),
	}
}

// calculateCost returns the USD cost for the given provider, model, and token counts.
// Returns 0.0 for unknown provider/model combinations.
func calculateCost(provider, model string, tokensIn, tokensOut uint64) float64 {
	provPricing, ok := pricingTable[provider]
	if !ok {
		return 0.0
	}
	pricing, ok := provPricing[model]
	if !ok {
		return 0.0
	}
	// pricing[0] = USD per 1M input tokens; pricing[1] = USD per 1M output tokens
	return float64(tokensIn)/1_000_000.0*pricing[0] + float64(tokensOut)/1_000_000.0*pricing[1]
}

// Track records one LLM invocation: it updates the ring buffer, the cumulative totals,
// and the Prometheus counter.
func (ct *CostTracker) Track(provider, model string, tokensIn, tokensOut uint64) {
	cost := calculateCost(provider, model, tokensIn, tokensOut)

	ct.mu.Lock()
	ct.entries[ct.head] = CostEntry{
		Provider:  provider,
		Model:     model,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostUSD:   cost,
	}
	ct.head = (ct.head + 1) % costRingSize
	if ct.size < costRingSize {
		ct.size++
	}

	key := provider + "/" + model
	if ct.totals[key] == nil {
		ct.totals[key] = &ModelCost{Provider: provider, Model: model}
	}
	ct.totals[key].TokensIn += tokensIn
	ct.totals[key].TokensOut += tokensOut
	ct.totals[key].CostUSD += cost
	ct.mu.Unlock()

	if cost > 0 {
		llmCostDollarsTotal.WithLabelValues(provider, model).Add(cost)
	}
}

// Summary returns cumulative totals plus a per-model breakdown.
func (ct *CostTracker) Summary() CostSummary {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	summary := CostSummary{
		ByModel: make([]ModelCost, 0, len(ct.totals)),
	}
	for _, m := range ct.totals {
		summary.TotalCostUSD += m.CostUSD
		summary.TotalTokensIn += m.TokensIn
		summary.TotalTokensOut += m.TokensOut
		summary.ByModel = append(summary.ByModel, *m)
	}
	return summary
}
