package bulkhead

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrBulkheadFull    = errors.New("bulkhead: pool is full, request rejected")
	ErrBulkheadTimeout = errors.New("bulkhead: timed out waiting for a worker slot")
	ErrBulkheadClosed  = errors.New("bulkhead: bulkhead is closed")
	ErrNilTask         = errors.New("bulkhead: task must not be nil")
)

type Task func(ctx context.Context) (interface{}, error)

type Bulkhead interface {
	Execute(ctx context.Context, task Task) (interface{}, error)
	TryExecute(ctx context.Context, task Task) (interface{}, error)
	Metrics() Metrics
	Name() string
	Close() error
}

type Metrics struct {
	Name           string
	MaxConcurrency int
	MaxQueueSize   int
	ActiveWorkers  int64
	QueuedRequests int64
	TotalExecuted  int64
	TotalSucceeded int64
	TotalFailed    int64
	TotalRejected  int64
	TotalTimedOut  int64
	AvgExecutionMs float64
	CreatedAt      time.Time
}

type Option func(*WorkerPoolBulkhead)

func WithMaxConcurrency(n int) Option {
	return func(b *WorkerPoolBulkhead) {
		if n > 0 {
			b.maxConcurrency = n
		}
	}
}

func WithMaxQueueSize(n int) Option {
	return func(b *WorkerPoolBulkhead) {
		if n >= 0 {
			b.maxQueueSize = n
		}
	}
}

func WithMaxWait(d time.Duration) Option {
	return func(b *WorkerPoolBulkhead) {
		b.maxWait = d
	}
}

func WithOnReject(fn func(name string)) Option {
	return func(b *WorkerPoolBulkhead) {
		b.onReject = fn
	}
}

func WithOnSuccess(fn func(name string, duration time.Duration)) Option {
	return func(b *WorkerPoolBulkhead) {
		b.onSuccess = fn
	}
}

func WithOnFailure(fn func(name string, err error)) Option {
	return func(b *WorkerPoolBulkhead) {
		b.onFailure = fn
	}
}

type WorkerPoolBulkhead struct {
	name           string
	maxConcurrency int
	maxQueueSize   int
	maxWait        time.Duration
	sem            chan struct{}
	queue          chan *job
	once           sync.Once
	closed         chan struct{}
	wg             sync.WaitGroup
	onReject       func(name string)
	onSuccess      func(name string, duration time.Duration)
	onFailure      func(name string, err error)
	activeWorkers  int64
	queuedRequests int64
	totalExecuted  int64
	totalSucceeded int64
	totalFailed    int64
	totalRejected  int64
	totalTimedOut  int64
	totalExecNs    int64
	createdAt      time.Time
}

type job struct {
	ctx    context.Context
	task   Task
	result chan jobResult
}

type jobResult struct {
	value interface{}
	err   error
}

func New(name string, opts ...Option) *WorkerPoolBulkhead {
	b := &WorkerPoolBulkhead{
		name:           name,
		maxConcurrency: 10,
		maxQueueSize:   100,
		closed:         make(chan struct{}),
		createdAt:      time.Now(),
	}
	for _, o := range opts {
		o(b)
	}

	b.sem = make(chan struct{}, b.maxConcurrency)
	for i := 0; i < b.maxConcurrency; i++ {
		b.sem <- struct{}{}
	}

	if b.maxQueueSize > 0 {
		b.queue = make(chan *job, b.maxQueueSize)
		b.wg.Add(1)
		go b.dispatch()
	}

	return b
}

func (b *WorkerPoolBulkhead) dispatch() {
	defer b.wg.Done()
	for {
		select {
		case <-b.closed:
			for {
				select {
				case j := <-b.queue:
					atomic.AddInt64(&b.queuedRequests, -1)
					j.result <- jobResult{nil, ErrBulkheadClosed}
				default:
					return
				}
			}
		case j := <-b.queue:
			atomic.AddInt64(&b.queuedRequests, -1)
			select {
			case <-b.sem:
				b.wg.Add(1)
				go b.run(j)
			case <-b.closed:
				j.result <- jobResult{nil, ErrBulkheadClosed}
			}
		}
	}
}

func (b *WorkerPoolBulkhead) run(j *job) {
	defer b.wg.Done()
	defer func() { b.sem <- struct{}{} }()

	atomic.AddInt64(&b.activeWorkers, 1)
	defer atomic.AddInt64(&b.activeWorkers, -1)

	start := time.Now()
	value, err := j.task(j.ctx)
	elapsed := time.Since(start)

	atomic.AddInt64(&b.totalExecuted, 1)
	atomic.AddInt64(&b.totalExecNs, elapsed.Nanoseconds())

	if err != nil {
		atomic.AddInt64(&b.totalFailed, 1)
		if b.onFailure != nil {
			b.onFailure(b.name, err)
		}
	} else {
		atomic.AddInt64(&b.totalSucceeded, 1)
		if b.onSuccess != nil {
			b.onSuccess(b.name, elapsed)
		}
	}

	j.result <- jobResult{value, err}
}

func (b *WorkerPoolBulkhead) Execute(ctx context.Context, task Task) (interface{}, error) {
	if task == nil {
		return nil, ErrNilTask
	}
	select {
	case <-b.closed:
		return nil, ErrBulkheadClosed
	default:
	}

	j := &job{ctx: ctx, task: task, result: make(chan jobResult, 1)}

	if b.maxQueueSize > 0 {
		return b.enqueue(ctx, j)
	}
	return b.acquireAndRun(ctx, j)
}

func (b *WorkerPoolBulkhead) TryExecute(ctx context.Context, task Task) (interface{}, error) {
	if task == nil {
		return nil, ErrNilTask
	}
	select {
	case <-b.closed:
		return nil, ErrBulkheadClosed
	default:
	}

	select {
	case <-b.sem:
	default:
		atomic.AddInt64(&b.totalRejected, 1)
		if b.onReject != nil {
			b.onReject(b.name)
		}
		return nil, ErrBulkheadFull
	}

	j := &job{ctx: ctx, task: task, result: make(chan jobResult, 1)}
	b.wg.Add(1)
	go b.run(j)

	select {
	case res := <-j.result:
		return res.value, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *WorkerPoolBulkhead) enqueue(ctx context.Context, j *job) (interface{}, error) {
	waitCtx := ctx
	var cancel context.CancelFunc
	if b.maxWait > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, b.maxWait)
		defer cancel()
	}

	select {
	case b.queue <- j:
		atomic.AddInt64(&b.queuedRequests, 1)
	default:
		// Queue is full → reject immediately.
		atomic.AddInt64(&b.totalRejected, 1)
		if b.onReject != nil {
			b.onReject(b.name)
		}
		return nil, ErrBulkheadFull
	}

	select {
	case res := <-j.result:
		return res.value, res.err
	case <-waitCtx.Done():
		atomic.AddInt64(&b.totalTimedOut, 1)
		if waitCtx.Err() == context.DeadlineExceeded && b.maxWait > 0 {
			return nil, ErrBulkheadTimeout
		}
		return nil, ctx.Err()
	}
}

func (b *WorkerPoolBulkhead) acquireAndRun(ctx context.Context, j *job) (interface{}, error) {
	waitCtx := ctx
	var cancel context.CancelFunc
	if b.maxWait > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, b.maxWait)
		defer cancel()
	}

	select {
	case <-b.sem:
	case <-waitCtx.Done():
		atomic.AddInt64(&b.totalRejected, 1)
		atomic.AddInt64(&b.totalTimedOut, 1)
		if b.onReject != nil {
			b.onReject(b.name)
		}
		if b.maxWait > 0 && waitCtx.Err() == context.DeadlineExceeded {
			return nil, ErrBulkheadTimeout
		}
		return nil, ctx.Err()
	case <-b.closed:
		return nil, ErrBulkheadClosed
	}

	b.wg.Add(1)
	go b.run(j)

	select {
	case res := <-j.result:
		return res.value, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *WorkerPoolBulkhead) Metrics() Metrics {
	exec := atomic.LoadInt64(&b.totalExecuted)
	var avgMs float64
	if exec > 0 {
		avgMs = float64(atomic.LoadInt64(&b.totalExecNs)) / float64(exec) / 1e6
	}
	return Metrics{
		Name:           b.name,
		MaxConcurrency: b.maxConcurrency,
		MaxQueueSize:   b.maxQueueSize,
		ActiveWorkers:  atomic.LoadInt64(&b.activeWorkers),
		QueuedRequests: atomic.LoadInt64(&b.queuedRequests),
		TotalExecuted:  exec,
		TotalSucceeded: atomic.LoadInt64(&b.totalSucceeded),
		TotalFailed:    atomic.LoadInt64(&b.totalFailed),
		TotalRejected:  atomic.LoadInt64(&b.totalRejected),
		TotalTimedOut:  atomic.LoadInt64(&b.totalTimedOut),
		AvgExecutionMs: avgMs,
		CreatedAt:      b.createdAt,
	}
}

func (b *WorkerPoolBulkhead) Name() string { return b.name }

func (b *WorkerPoolBulkhead) Close() error {
	var err error
	b.once.Do(func() {
		close(b.closed)
		b.wg.Wait()
	})
	return err
}

func (b *WorkerPoolBulkhead) String() string {
	m := b.Metrics()
	return fmt.Sprintf(
		"Bulkhead(%s) workers=%d/%d queued=%d exec=%d ok=%d fail=%d rejected=%d",
		m.Name, m.ActiveWorkers, m.MaxConcurrency,
		m.QueuedRequests, m.TotalExecuted,
		m.TotalSucceeded, m.TotalFailed, m.TotalRejected,
	)
}
