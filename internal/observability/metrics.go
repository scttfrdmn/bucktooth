package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MessageDurationSeconds tracks end-to-end message processing latency.
	MessageDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "bucktooth_message_duration_seconds",
		Help:    "End-to-end message processing latency in seconds.",
		Buckets: prometheus.DefBuckets,
	})

	// MessagesTotal counts messages by direction ("in" or "out").
	MessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bucktooth_messages_total",
		Help: "Total number of messages by direction.",
	}, []string{"direction"})

	// TokensTotal counts LLM tokens consumed by direction ("in" or "out").
	TokensTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bucktooth_tokens_total",
		Help: "Total LLM tokens consumed by direction.",
	}, []string{"direction"})

	// ActiveUsers tracks the number of users with live agent instances.
	ActiveUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "bucktooth_active_users",
		Help: "Number of users with active agent instances.",
	})
)
