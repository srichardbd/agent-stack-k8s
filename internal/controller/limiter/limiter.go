package limiter

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/buildkite/agent-stack-k8s/v2/internal/controller/model"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// MaxInFlight is a job handler that wraps another job handler
// (typically the actual job scheduler) and only creates new jobs if the total
// number of jobs currently running is below a limit.
type MaxInFlight struct {
	// MaxInFlight sets the upper limit on number of jobs running concurrently
	// in the cluster. 0 means no limit.
	MaxInFlight int

	// Next handler in the chain.
	handler model.JobHandler

	// Logs go here
	logger *zap.Logger

	// When a job starts, it takes a token from the bucket.
	// When a job ends, it puts a token back in the bucket.
	tokenBucket chan struct{}
}

// New creates a MaxInFlight limiter. maxInFlight must be at least 1.
func New(logger *zap.Logger, scheduler model.JobHandler, maxInFlight int) *MaxInFlight {
	if maxInFlight <= 0 {
		// Using panic, because getting here is severe programmer error and the
		// whole controller is still just starting up.
		panic(fmt.Sprintf("maxInFlight <= 0 (got %d)", maxInFlight))
	}
	maxInFlightGauge.Set(float64(maxInFlight))
	l := &MaxInFlight{
		handler:     scheduler,
		MaxInFlight: maxInFlight,
		logger:      logger,
		tokenBucket: make(chan struct{}, maxInFlight),
	}
	for range maxInFlight {
		// Fill the bucket with tokens.
		l.tokenBucket <- struct{}{}
	}
	// Rather than calling gauge.Set, get the number of tokens during scrape.
	tokensAvailableFunc = func() int { return len(l.tokenBucket) }
	return l
}

// RegisterInformer registers the limiter to listen for Kubernetes job events,
// and waits for cache sync.
func (l *MaxInFlight) RegisterInformer(ctx context.Context, factory informers.SharedInformerFactory) error {
	informer := factory.Batch().V1().Jobs()
	jobInformer := informer.Informer()
	reg, err := jobInformer.AddEventHandler(l)
	if err != nil {
		return err
	}
	go factory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), reg.HasSynced) {
		return fmt.Errorf("failed to sync informer cache")
	}

	return nil
}

// Handle either passes the job onto the next handler immediately, or blocks
// until there is capacity. It returns [model.ErrStaleJob] if the job data
// becomes too stale while waiting for capacity.
func (l *MaxInFlight) Handle(ctx context.Context, job model.Job) error {
	// Block until there's a token in the bucket, or cancel if the job
	// information becomes too stale.
	start := time.Now()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)

	case <-job.StaleCh:
		return model.ErrStaleJob

	case <-l.tokenBucket:
		l.logger.Debug("token acquired",
			zap.String("uuid", job.Uuid),
			zap.Int("available-tokens", len(l.tokenBucket)),
		)
	}
	tokenWaitDurationHistogram.Observe(time.Since(start).Seconds())

	// We got a token from the bucket above! Proceed to schedule the pod.
	// The next handler should be Scheduler (except in some tests).
	l.logger.Debug("passing job to next handler",
		zap.Stringer("handler", reflect.TypeOf(l.handler)),
		zap.String("uuid", job.Uuid),
	)
	if err := l.handler.Handle(ctx, job); err != nil {
		// Oh well. Return the token.
		l.tryReturnToken()

		l.logger.Debug("next handler failed",
			zap.String("uuid", job.Uuid),
			zap.Int("available-tokens", len(l.tokenBucket)),
		)
		return err
	}
	return nil
}

// OnAdd is called by k8s to inform us a resource is added.
func (l *MaxInFlight) OnAdd(obj any, inInitialList bool) {
	onAddEvents.Inc()
	job, _ := obj.(*batchv1.Job)
	if job == nil {
		return
	}
	if !inInitialList {
		// After sync is finished, the limiter handler takes tokens directly.
		return
	}
	// Before sync is finished: we're learning about existing jobs, so we should
	// (try to) take tokens for unfinished jobs started by a previous controller.
	// If it's added as already finished, no need to take a token for it.
	// Otherwise, try to take one, but don't block (in case the stack was
	// restarted with a different limit).
	if !model.JobFinished(job) {
		l.tryTakeToken()
	}
	l.logger.Debug("at end of OnAdd", zap.Int("tokens-available", len(l.tokenBucket)))
}

// OnUpdate is called by k8s to inform us a resource is updated.
func (l *MaxInFlight) OnUpdate(prev, curr any) {
	onUpdateEvents.Inc()
	prevState, _ := prev.(*batchv1.Job)
	currState, _ := curr.(*batchv1.Job)
	if prevState == nil || currState == nil {
		return
	}
	// Only return a token if the job state has *changed* from not-finished to
	// finished.
	if !model.JobFinished(prevState) && model.JobFinished(currState) {
		l.tryReturnToken()
		l.logger.Debug("job state changed from not-finished to finished", zap.Int("tokens-available", len(l.tokenBucket)))
	}
}

// OnDelete is called by k8s to inform us a resource is deleted.
func (l *MaxInFlight) OnDelete(obj any) {
	onDeleteEvents.Inc()
	prevState, _ := obj.(*batchv1.Job)
	if prevState == nil {
		return
	}

	// OnDelete gives us the last-known state prior to deletion.
	// If that state was finished, we've already returned a token.
	// If that state was not-finished, we need to return a token now.
	if !model.JobFinished(prevState) {
		l.tryReturnToken()
	}
	l.logger.Debug("at end of OnDelete", zap.Int("tokens-available", len(l.tokenBucket)))
}

// tryTakeToken takes a token from the bucket, if one is available. It does not
// block.
func (l *MaxInFlight) tryTakeToken() {
	select {
	case <-l.tokenBucket:
	default:
	}
}

// tryReturnToken returns a token to the bucket, if not full. It does not block.
func (l *MaxInFlight) tryReturnToken() {
	select {
	case l.tokenBucket <- struct{}{}:
	default:
	}
}
