package cronna

import "context"

type CronnaJob struct {
	jobs []func(ctx context.Context) error
}

func (cr *CronnaJob) Start() {}

func (cr *CronnaJob) AddFunc(expresion string, fn func(ctx context.Context) error) {

}

// parser
// matcher
// runner
