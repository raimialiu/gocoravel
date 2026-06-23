package scheduler

import (
	"context"
)

type Executor struct {
	scheuler    *Scheduler
	_cancelFunc context.CancelFunc
}
