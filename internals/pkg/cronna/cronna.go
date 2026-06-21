package cronna

import (
	"context"

	"github.com/raimialiu/gocoravel/internals/pkg/collections/dictionary/concurrent_dictionary"
)

type (
	CronnaJob struct {
		jobs    concurrent_dictionary.ConcurrentDictionary[string, Job]
		_runner Runner
	}
)

func (cr *CronnaJob) Start(ctx context.Context) error {
	jobs := cr.jobs.ToMap()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for _, job := range jobs {
					runner := NewRunner(job)
					runner.Run(ctx, nil)
				}
			}
		}
	}()

	return nil
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
