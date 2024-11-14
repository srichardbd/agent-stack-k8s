package limiter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	promNamespace = "buildkite_agent_stack_k8s"
	promSubsystem = "limiter"
)

// Overridden by New to return len(tokenBucket).
var tokensAvailableFunc = func() int { return 0 }

var (
	maxInFlightGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "max_in_flight",
		Help:      "Configured limit on number of jobs simultaneously in flight",
	})
	_ = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "tokens_available",
		Help:      "Limiter tokens available",
	}, func() float64 { return float64(tokensAvailableFunc()) })
	tokenWaitDurationHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                    promNamespace,
		Subsystem:                    promSubsystem,
		Name:                         "token_wait_duration",
		Help:                         "Time spent waiting for a limiter token to become available",
		NativeHistogramBucketFactor:  1.1,
		NativeHistogramZeroThreshold: 0.01,
	})

	onAddEvents = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "onadd_events",
		Help:      "Count of OnAdd informer events",
	})
	onUpdateEvents = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "onupdate_events",
		Help:      "Count of OnUpdate informer events",
	})
	onDeleteEvents = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "ondelete_events",
		Help:      "Count of OnDelete informer events",
	})
)
