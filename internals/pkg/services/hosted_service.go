package services

import (
	"context"

	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
)

func NewHostedService() *HostedService {
	return &HostedService{}
}

func (h *HostedService) LoopForever(ctx context.Context, jobs chan []scheduler.ScheduleTask) {
	defer close(jobs)
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
