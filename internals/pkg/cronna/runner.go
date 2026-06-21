package cronna

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Runner struct {
	expression *Expression
}

func NewRunner(expression *Expression) *Runner {
	return &Runner{
		expression: expression,
	}
}

func (r *Runner) NextTime(from time.Time) (time.Time, error) {
	// start from the next minute — `from` itself is excluded
	t := from.Truncate(time.Minute).Add(time.Minute)

	limit := from.Add(4 * 365 * 24 * time.Hour)

	for t.Before(limit) {
		if r.expression.Matches(t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("cronna: no next time found for expression %q within 4 years", r.expression._rawExpression)
}

func (r *Runner) Run(ctx context.Context, job func(ctx context.Context)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			next, err := r.NextTime(time.Now())
			if err != nil {
				log.Printf("cronna: error getting next time for %q: %v", r.expression._rawExpression, err)
				return
			} else {
				time.Sleep(time.Until(next))
			}

			go job(ctx)
		}
	}
}
