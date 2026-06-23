package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/raimialiu/gocoravel/internals/pkg/collections/dictionary/concurrent_dictionary"
	"github.com/raimialiu/gocoravel/internals/pkg/system/delegate"
	"github.com/raimialiu/gostream/stream"
	_ "github.com/raimialiu/gostream/stream"
)

type (
	Scheduler struct {
		_tasks        *concurrent_dictionary.ConcurrentDictionary[string, ScheduleEvent]
		_ctx          context.Context
		_errorHandler delegate.Action[error]
		_cancel       context.CancelFunc
	} // hold job list, and begin the whole process together

	ScheduleOpts func(asyncFunc *delegate.ActionOrAsyncFunc)
)

func (s *Scheduler) Schedule(
	invocableType *IInvocable,
	options ...ScheduleOpts,
) *ScheduleEvent {
	function := &delegate.ActionOrAsyncFunc{}
	for _, option := range options {
		option(function)
	}

	ev := NewScheduleEvent()
	if invocableType != nil {
		ev._invocableType = invocableType
	}

	s._tasks.TryAdd(ev.OverlappingUniqueIdentifier(), *ev)
	return ev
}

func (s *Scheduler) ScheduleSimple(actionToSchedule delegate.Action[interface{}]) *ScheduleEvent {
	event := NewScheduleEvent(
		WithScheduleAction[interface{}](actionToSchedule),
	)

	s._tasks.TryAdd(event.OverlappingUniqueIdentifier(), *event)
	return event
}

func (s *Scheduler) SchedulePureFunc(function func(params ...any) any, params ...any) *ScheduleEvent {
	event := NewScheduleEvent(
		WithInvocableTypeAndParams(function, params...),
	)

	s._tasks.TryAdd(event.OverlappingUniqueIdentifier(), *event)
	return event
}

func (s *Scheduler) ScheduleFunc(fun delegate.Func[interface{}, interface{}]) *ScheduleEvent {
	event := NewScheduleEvent(
		WithScheduleFunc[interface{}](fun),
	)

	s._tasks.TryAdd(event.OverlappingUniqueIdentifier(), *event)
	return event
}

func (s *Scheduler) ScheduleInvocable(invocableType any) *ScheduleEvent {
	if _, ok := invocableType.(IInvocable); !ok {
		panic("invalid invocable type")
	}

	event := NewScheduleEvent(
		WithInvocableType(invocableType),
	)

	s._tasks.TryAdd(event.OverlappingUniqueIdentifier(), *event)
	return event
}

func (s *Scheduler) ScheduleInvocableWithParams(invocableType any, params ...interface{}) *ScheduleEvent {
	if _, ok := invocableType.(IInvocable); !ok {
		panic("invalid invocable type")
	}

	event := NewScheduleEvent(
		WithInvocableTypeAndParams(invocableType, params),
	)

	s._tasks.TryAdd(event.OverlappingUniqueIdentifier(), *event)
	return event
}

func NewScheduler(ctx context.Context) *Scheduler {
	return &Scheduler{
		_tasks: concurrent_dictionary.New[string, ScheduleEvent](nil, nil),
		_ctx:   ctx,
	}
}

func (s *Scheduler) RunAt(time time.Time) *Scheduler {
	var mt sync.Mutex
	mt.Lock()
	defer mt.Unlock()
	go s.runJobs(time)

	return s
}

func (s *Scheduler) runJobs(t time.Time) {
	jobs := s._tasks.ToMap()
	scheduledJobs := make([]ScheduleEvent, 0)
	for _, job := range jobs {
		timerIsAtMinute := t.Second() == 0
		taskIsSecondsBased := !job.IsCronBasedTask()
		runOnceAtStart := job.RunOnceAtStart()
		canRunBasedOnTimeMarker := timerIsAtMinute || taskIsSecondsBased

		if canRunBasedOnTimeMarker && job.IsDue(t) || runOnceAtStart {
			scheduledJobs = append(scheduledJobs, job)
			go func() {
				job.InvokeScheduledEvent(t)
			}()
		}

	}

	groupedJobs := stream.Of(scheduledJobs...).
		GroupBy(func(event ScheduleEvent) interface{} {
			return event.OverlappingUniqueIdentifier()
		})

	for _, groupJobs := range groupedJobs {
		for _, job := range groupJobs {
			go func() {
				job.InvokeScheduledEvent(t)
			}()
		}
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
