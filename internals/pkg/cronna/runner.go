package cronna

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Runner struct {
	job Job
}

func NewRunner(job Job) *Runner {
	return &Runner{
		job: job,
	}
}

func (r *Runner) NextTime(from time.Time) (time.Time, error) {
	// start from the next minute — `from` itself is excluded
	t := from.Truncate(time.Minute).Add(time.Minute)

	limit := from.Add(4 * 365 * 24 * time.Hour)

	for t.Before(limit) {
		if r.job.expression.Matches(t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("cronna: no next time found for expression %q within 4 years", r.job.expression._rawExpression)
}

func (r *Runner) Run(ctx context.Context, job *func(ctx context.Context) error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			next, err := r.NextTime(time.Now())
			if err != nil {
				log.Printf("cronna: error getting next time for %q: %v", r.job.expression._rawExpression, err)
				return
			} else {
				time.Sleep(time.Until(next))
			}

			if job == nil {
				go func() {
					err := r.job.fn(ctx)
					if err != nil {
						log.Printf("cronna: error running job %q: %v", r.job.expression._rawExpression, err)
						panic(err)
					}
				}()
			} else {
				fn := *job
				go fn(ctx)
			}
		}
	}
}
