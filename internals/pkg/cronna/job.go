package cronna

import (
	"context"
	"time"

	"github.com/raimialiu/gocoravel/internals/pkg/collections/dictionary/concurrent_dictionary"
)

type (
	Ijob interface {
		Name() string
		CronExpression() string
		Fn(ctx context.Context) error
		Instance() interface{}
	}
	Job struct {
		name          string
		expression    Expression
		rawExpression string
		_nextRun      time.Time
		fn            func(ctx context.Context) error
	}
)

func (j Job) Name() string {
	return j.name
}

func (j Job) Instance() interface{} {
	return j
}

func (j Job) CronExpression() string {
	return j.rawExpression
}

func (j Job) Expression() Expression {
	return j.expression
}

func (j Job) Next(t time.Time) time.Time {
	tr := t.Truncate(time.Minute).Add(time.Minute)
	return tr
}

func (j Job) Fn(ctx context.Context) error {
	go func() {
		err := j.fn(ctx)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

func NewJob(cron string, fn func(ctx context.Context) error) *Job {
	fnName := concurrent_dictionary.FuncName(fn)
	expression := *NewExpression(cron)

	return &Job{
		name:       fnName,
		expression: expression,
		fn:         fn,
	}
}
