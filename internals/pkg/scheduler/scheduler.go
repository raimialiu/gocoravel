package scheduler

import (
	"context"

	"github.com/raimialiu/gocoravel/internals/pkg/collections/dictionary/concurrent_dictionary"
	"github.com/raimialiu/gocoravel/internals/pkg/system/delegate"
	_ "github.com/raimialiu/gostream/stream"
)

type (
	Scheduler struct {
		_tasks        *concurrent_dictionary.ConcurrentDictionary[string, ScheduleTask]
		_ctx          context.Context
		_errorHandler delegate.Action[error]
	} // hold job list, and begin the whole process together
)

func NewScheduler() *Scheduler {
	return &Scheduler{
		_tasks: concurrent_dictionary.New[string, ScheduleTask](nil, nil),
		_ctx:   context.Background(),
	}
}

func (s *Scheduler) CancelTasks() {
	if s._ctx.Err() != nil {
		_, cancel := context.WithCancelCause(s._ctx)
		defer cancel(s._ctx.Err())
	}
}

func (s *Scheduler) TryUnschedule(taskId string) bool {
	task := s._tasks.Get(taskId)
	if task == nil {
		return false
	}

	return s._tasks.TryRemove(taskId)
}
