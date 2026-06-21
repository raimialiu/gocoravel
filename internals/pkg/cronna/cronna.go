package cronna

import (
	"context"
	"time"

	"github.com/raimialiu/gocoravel/internals/pkg/collections/dictionary/concurrent_dictionary"
)

type (
	CronnaJob struct {
		jobs    *concurrent_dictionary.ConcurrentDictionary[string, Job]
		_runner Runner
		_ctx    context.Context
		_cancel context.CancelFunc
	}
)

func New() *CronnaJob {
	ctx, cancel := context.WithCancel(context.Background())
	cronna := &CronnaJob{
		jobs: concurrent_dictionary.New[string, Job](
			concurrent_dictionary.WithCapacity(23),
			concurrent_dictionary.WithConcurrencyLevel(4),
			concurrent_dictionary.WithLoadFactor(0.86),
		),
		_ctx:    ctx,
		_cancel: cancel,
	}

	return cronna
}

func (cr *CronnaJob) Start() error {

	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case t := <-ticker.C:
				cr.runDue(cr._ctx, t)
			case <-cr._ctx.Done():
				ticker.Stop()
			default:
				return
			}
		}
	}()

	return nil
}

func (cr *CronnaJob) runDue(ctx context.Context, t time.Time) {
	jobs := cr.jobs.ToMap()
	for _, job := range jobs {
		if job.expression.Matches(t) {
			runner := NewRunner(job)
			runner.Run(ctx, nil)
		}
	}
}

func (cr *CronnaJob) Stop() {
	cr._cancel()
}

func (cr *CronnaJob) AddFunc(cron string, fn func(ctx context.Context) error) {
	job := NewJob(cron, fn)
	fnName := concurrent_dictionary.FuncName(job.Name())
	cr.jobs.TryAdd(fnName, *job)
}

func (cr *CronnaJob) AddJob(job Ijob) {
	instance := job.(*Job)
	cr.jobs.TryAdd(job.Name(), *instance)
}

// parser => done
// matcher => done
// runner
