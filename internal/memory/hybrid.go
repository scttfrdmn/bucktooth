package memory

import (
	"context"
	"math"
	"sort"
	"time"

	agenmem "github.com/scttfrdmn/agenkit/agenkit-go/memory"
)

// HybridStore combines BM25 keyword ranking and vector cosine similarity.
// Messages are written to both an in-memory raw store (for BM25 scoring) and a
// vector store (for semantic embeddings). GetHistory returns messages ranked by a
// weighted combination of both signals.
type HybridStore struct {
	raw           *InMemoryStore
	vec           *VectorStore
	bm25Weight    float64 // 0.0 = pure semantic, 1.0 = pure BM25
	scorer        BM25Scorer
	decayEnabled  bool
	decayHalfLife float64 // hours; must be > 0 when decayEnabled
}

// NewHybridStore creates a HybridStore. hybridWeight controls the blend
// (0.0–1.0 clamped); embedProvider is used by the underlying VectorStore.
// When decayEnabled is true, recency is scored via exponential decay with
// the given half-life in hours (must be > 0; falls back to linear if 0).
func NewHybridStore(embedProvider agenmem.EmbeddingProvider, hybridWeight float64, decayEnabled bool, decayHalfLifeHours float64) *HybridStore {
	if hybridWeight < 0 {
		hybridWeight = 0
	}
	if hybridWeight > 1 {
		hybridWeight = 1
	}
	return &HybridStore{
		raw:           NewInMemoryStore(),
		vec:           NewVectorStore(embedProvider),
		bm25Weight:    hybridWeight,
		scorer:        BM25Scorer{},
		decayEnabled:  decayEnabled,
		decayHalfLife: decayHalfLifeHours,
	}
}

// AddMessage stores a message in both the raw and vector stores.
func (h *HybridStore) AddMessage(ctx context.Context, userID string, msg Message) error {
	if err := h.raw.AddMessage(ctx, userID, msg); err != nil {
		return err
	}
	return h.vec.AddMessage(ctx, userID, msg)
}

// GetHistory returns up to limit messages, ranked by a hybrid BM25 + recency
// score. The most recent message in the history serves as the implicit query for
// BM25 scoring. When there are fewer messages than limit, all messages are
// returned in original order.
func (h *HybridStore) GetHistory(ctx context.Context, userID string, limit int) ([]Message, error) {
	// Fetch all messages from the raw store (up to a generous cap).
	all, err := h.raw.GetHistory(ctx, userID, 1000)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 || len(all) <= limit {
		return all, nil
	}

	// Use the content of the most recent message as the BM25 query.
	query := tokenize(all[len(all)-1].Content)

	// Build the tokenised corpus.
	corpus := make([][]string, len(all))
	for i, m := range all {
		corpus[i] = tokenize(m.Content)
	}

	// Compute BM25 scores and normalise to [0, 1].
	bm25Raw := h.scorer.Score(corpus, query)
	bm25Norm := normalise(bm25Raw)

	// Recency score: more recent messages score higher.
	// With decay enabled: exponential decay (exp(-age * ln2 / halfLife)); score=1 at age=0, 0.5 at age=halfLife.
	// Without decay (or invalid halfLife): linear 0→1, guarded against len==1 divide-by-zero.
	recency := make([]float64, len(all))
	if h.decayEnabled && h.decayHalfLife > 0 {
		now := time.Now()
		for i, m := range all {
			ageHours := now.Sub(m.Timestamp).Hours()
			if ageHours < 0 {
				ageHours = 0
			}
			recency[i] = math.Exp(-ageHours * math.Ln2 / h.decayHalfLife)
		}
	} else {
		n := len(all) - 1
		for i := range all {
			if n > 0 {
				recency[i] = float64(i) / float64(n)
			} else {
				recency[i] = 1
			}
		}
	}

	// Combine BM25 and recency signals.
	type scored struct {
		idx   int
		score float64
	}
	ranked := make([]scored, len(all))
	for i := range all {
		ranked[i] = scored{
			idx:   i,
			score: h.bm25Weight*bm25Norm[i] + (1-h.bm25Weight)*recency[i],
		}
	}
	sort.Slice(ranked, func(a, b int) bool {
		return ranked[a].score > ranked[b].score
	})

	if limit > len(all) {
		limit = len(all)
	}
	result := make([]Message, limit)
	for i := 0; i < limit; i++ {
		result[i] = all[ranked[i].idx]
	}
	return result, nil
}

// ClearHistory removes all stored messages for a user from both stores.
func (h *HybridStore) ClearHistory(ctx context.Context, userID string) error {
	if err := h.raw.ClearHistory(ctx, userID); err != nil {
		return err
	}
	return h.vec.ClearHistory(ctx, userID)
}

// Close closes both underlying stores.
func (h *HybridStore) Close() error {
	if err := h.raw.Close(); err != nil {
		return err
	}
	return h.vec.Close()
}

// normalise linearly scales a float64 slice to [0, 1].
// Returns the original slice unchanged if all values are equal.
func normalise(scores []float64) []float64 {
	if len(scores) == 0 {
		return scores
	}
	minV, maxV := scores[0], scores[0]
	for _, s := range scores[1:] {
		if s < minV {
			minV = s
		}
		if s > maxV {
			maxV = s
		}
	}
	if maxV == minV {
		return scores
	}
	out := make([]float64, len(scores))
	for i, s := range scores {
		out[i] = (s - minV) / (maxV - minV)
	}
	return out
}
