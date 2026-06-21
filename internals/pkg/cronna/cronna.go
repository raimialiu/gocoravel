package cronna

import (
	"context"
	"fmt"
	"time"

	"github.com/raimialiu/gocoravel/internals/pkg/collections/dictionary/concurrent_dictionary"
	"github.com/raimialiu/gostream/stream"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
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
				return
			}

			fmt.Printf("%scronna :-> heartbeat%s\n", colorCyan, colorReset)
		}
	}()

	return nil
}

func (cr *CronnaJob) runDue(ctx context.Context, t time.Time) {
	jobs := stream.FromMap(cr.jobs.ToMap()).
		Filter(func(k stream.KeyValue[interface{}, Job]) bool {
			return !k.Value._nextRun.After(t)
		}).ToList()
	for _, job := range jobs {
		if job.Value.expression.Matches(t) {
			runner := NewRunner(job.Value)
			go runner.Run(ctx, nil)
			job.Value._nextRun = job.Value.Next(t)
			cr.jobs.TryUpdate(job.Key.(string), job.Value)
		}
	}

	fmt.Println("cronna: jobs done")
}

func (cr *CronnaJob) Stop() {
	cr._cancel()
}

func (cr *CronnaJob) AddFunc(cron string, fn func() error) {
	jobFn := func(ctx context.Context) error {
		return fn()
	}
	job := NewJob(cron, jobFn)
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
