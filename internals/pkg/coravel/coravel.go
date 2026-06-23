package coravel

import (
	"context"

	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
)

type CoravelConfig func(scheduler *Coravel)

func NewCoravel(configs ...CoravelConfig) *Coravel {
	coravel := &Coravel{}

	for _, config := range configs {
		config(coravel)
	}

	return coravel
}

func (c *Coravel) AddScheduler() *Coravel {
	ctx, cancel := context.WithCancel(context.Background())
	s := scheduler.NewScheduler(ctx, cancel)
	h := scheduler.NewSchedulerHost(s, ctx, cancel)

	go h.Start()
	c._scheduler = *s
	c._host = *h
	c._ctx = ctx
	return c
}

func (c *Coravel) Scheduler() *scheduler.Scheduler {
	return &c._scheduler
}

func (c *Coravel) Stop() {
	c._scheduler.CancelTasks()
}

func (c *Coravel) UseScheduler(configs ...CoravelConfig) {
	for _, config := range configs {
		config(c)
	}
}
