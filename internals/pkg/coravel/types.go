package coravel

import (
	"context"

	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
)

type (
	Coravel struct {
		_host      scheduler.SchedulerHost
		_scheduler scheduler.Scheduler
		_ctx       context.Context
	}
)
