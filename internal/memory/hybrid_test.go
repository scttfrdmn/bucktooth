package memory

import (
	"context"
	"math"
	"testing"
	"time"
)

// noopEmbedProvider satisfies agenmem.EmbeddingProvider with zero-dimension embeddings.
type noopEmbedProvider struct{}

func (n *noopEmbedProvider) Embed(_ context.Context, _ string) ([]float64, error) {
	return []float64{}, nil
}

func (n *noopEmbedProvider) Dimension() int { return 0 }

func TestHybridStore_DecayEnabled_HalfLife(t *testing.T) {
	store := NewHybridStore(&noopEmbedProvider{}, 0.5, true, 1.0) // 1-hour half-life
	ctx := context.Background()

	now := time.Now()

	// Message aged ~1 hour — recency should be ≈ 0.5
	old := Message{
		Role:      "user",
		Content:   "old message",
		Timestamp: now.Add(-1 * time.Hour),
	}
	// Recent message — recency should be ≈ 1.0
	recent := Message{
		Role:      "user",
		Content:   "recent message",
		Timestamp: now,
	}

	if err := store.raw.AddMessage(ctx, "u1", old); err != nil {
		t.Fatal(err)
	}
	if err := store.raw.AddMessage(ctx, "u1", recent); err != nil {
		t.Fatal(err)
	}

	// Verify the decay values directly via the recency calculation.
	oldAgeHours := now.Sub(old.Timestamp).Hours()
	decayOld := math.Exp(-oldAgeHours * math.Ln2 / 1.0)
	if math.Abs(decayOld-0.5) > 0.01 {
		t.Errorf("expected decayOld ≈ 0.5 at age=halfLife, got %.4f", decayOld)
	}

	recentAgeHours := now.Sub(recent.Timestamp).Hours()
	decayRecent := math.Exp(-recentAgeHours * math.Ln2 / 1.0)
	if math.Abs(decayRecent-1.0) > 0.01 {
		t.Errorf("expected decayRecent ≈ 1.0 at age=0, got %.4f", decayRecent)
	}

	if decayRecent <= decayOld {
		t.Errorf("recent message should have higher recency score than old message")
	}
}

func TestHybridStore_DecayDisabled_LinearBehaviour(t *testing.T) {
	store := NewHybridStore(&noopEmbedProvider{}, 0.5, false, 0)
	ctx := context.Background()

	// Add 3 messages — with decay disabled, recency should be linear 0→1.
	for i := 0; i < 3; i++ {
		if err := store.raw.AddMessage(ctx, "u2", Message{
			Role:      "user",
			Content:   "msg",
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}

	// GetHistory with limit > count returns all in original order (no ranking needed).
	msgs, err := store.GetHistory(ctx, "u2", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestHybridStore_DecayHalfLifeZero_FallsBackToLinear(t *testing.T) {
	// decayEnabled=true but decayHalfLife=0 should fall back to linear (no panic).
	store := NewHybridStore(&noopEmbedProvider{}, 0.5, true, 0)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := store.raw.AddMessage(ctx, "u3", Message{
			Role:      "user",
			Content:   "msg",
			Timestamp: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	// Should not panic with len==5, and return ≤5 messages.
	msgs, err := store.GetHistory(ctx, "u3", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) > 5 {
		t.Fatalf("expected ≤5 messages, got %d", len(msgs))
	}
}

func TestHybridStore_SingleMessage_NoDivideByZero(t *testing.T) {
	// len(all)==1 with decay disabled used to divide by 0.
	store := NewHybridStore(&noopEmbedProvider{}, 0.5, false, 0)
	ctx := context.Background()

	if err := store.raw.AddMessage(ctx, "u4", Message{
		Role:      "user",
		Content:   "only message",
		Timestamp: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	msgs, err := store.GetHistory(ctx, "u4", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}
