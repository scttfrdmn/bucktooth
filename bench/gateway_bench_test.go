package bench_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/scttfrdmn/bucktooth/internal/channels"
	"github.com/scttfrdmn/bucktooth/internal/config"
	"github.com/scttfrdmn/bucktooth/internal/gateway"
	"github.com/scttfrdmn/bucktooth/internal/memory"
)

// BenchmarkEventBusPublish measures single-subscriber publish throughput.
func BenchmarkEventBusPublish(b *testing.B) {
	logger := zerolog.Nop()
	eb := gateway.NewEventBus(logger)

	eb.Subscribe(gateway.EventTypeMessageReceived, func(_ context.Context, _ gateway.Event) {})

	ev := gateway.MessageReceivedEvent(&channels.Message{
		ChannelID: "discord",
		UserID:    "user1",
		Content:   "hello world",
		Timestamp: time.Now(),
	})
	ctx := context.Background()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			eb.Publish(ctx, ev)
		}
	})
}

// BenchmarkEventBusPublishFanout measures 10-subscriber publish throughput.
func BenchmarkEventBusPublishFanout(b *testing.B) {
	logger := zerolog.Nop()
	eb := gateway.NewEventBus(logger)

	for i := 0; i < 10; i++ {
		eb.Subscribe(gateway.EventTypeMessageReceived, func(_ context.Context, _ gateway.Event) {})
	}

	ev := gateway.MessageReceivedEvent(&channels.Message{
		ChannelID: "discord",
		UserID:    "user1",
		Content:   "hello world",
		Timestamp: time.Now(),
	})
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		eb.Publish(ctx, ev)
	}
}

// BenchmarkInMemoryStoreAdd measures AddMessage throughput.
func BenchmarkInMemoryStoreAdd(b *testing.B) {
	store := memory.NewInMemoryStore()
	msg := memory.Message{
		Role:      "user",
		Content:   "benchmark message",
		Timestamp: time.Now(),
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.AddMessage(ctx, "user1", msg)
		}
	})
}

// BenchmarkInMemoryStoreGet measures GetHistory throughput on a pre-seeded store.
func BenchmarkInMemoryStoreGet(b *testing.B) {
	store := memory.NewInMemoryStore()
	ctx := context.Background()

	// Pre-seed 100 messages
	for i := 0; i < 100; i++ {
		_ = store.AddMessage(ctx, "user1", memory.Message{
			Role:      "user",
			Content:   "seeded message",
			Timestamp: time.Now(),
		})
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.GetHistory(ctx, "user1", 20)
		}
	})
}

// BenchmarkHTTPServerNew measures the allocation cost of NewHTTPServer.
func BenchmarkHTTPServerNew(b *testing.B) {
	logger := zerolog.Nop()
	registry := channels.NewChannelRegistry()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = gateway.NewHTTPServer(8080, registry, nil, gateway.NewStats(), logger)
	}
}

// BenchmarkConfigLoad measures YAML parse with defaults only (no file).
func BenchmarkConfigLoad(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = config.Load("")
	}
}
