package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	promNamespace = "buildkite_agent_stack_k8s"
	promSubsystem = "monitor"
)

var (
	jobQueryCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "job_queries",
		Help:      "Count of queries to Buildkite to fetch jobs",
	})
	jobQueryErrorCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "job_query_errors",
		Help:      "Count of errors from queries to Buildkite to fetch jobs",
	})
	jobQueryDurationHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace:                    promNamespace,
		Subsystem:                    promSubsystem,
		Name:                         "job_query_duration",
		Help:                         "Time taken to fetch jobs from Buildkite in seconds",
		NativeHistogramBucketFactor:  1.1,
		NativeHistogramZeroThreshold: 0.001,
	})
	jobsReturnedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "jobs_returned",
		Help:      "Count of jobs returned from queries to Buildkite",
	})
	jobsReachedWorkerCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "jobs_reached_worker",
		Help:      "Count of jobs received by a jobHandlerWorker",
	})
	jobsFilteredOutCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "jobs_filtered_out",
		Help:      "Count of jobs that didn't match the configured agent tags",
	})
	duplicateJobsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "duplicate_jobs",
		Help:      "Count of jobs that weren't scheduled because they were a duplicate of an existing job",
	})
	staleJobsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "stale_jobs",
		Help:      "Count of jobs that weren't scheduled because their information was queried too long ago",
	})
	jobHandlerErrorCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Subsystem: promSubsystem,
		Name:      "job_handler_errors",
		Help:      "Count of jobs that weren't scheduled because of some other handler error",
	})
)
