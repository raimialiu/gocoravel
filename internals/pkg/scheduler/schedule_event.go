package scheduler

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/adhocore/gronx"
	"github.com/raimialiu/gocoravel/internals/pkg/system/delegate"
)

const (
	ONE_MINUTE_AS_SECOND = 60
)

type (
	ScheduleEvent struct {
		_whenPredicate       delegate.Func[interface{}, bool]
		_parameters          []interface{}
		_scheduleAction      delegate.ActionOrAsyncFunc
		_runOnce             bool
		_wasPreviouslyRun    bool
		_preventOverlapping  bool
		_zoneTime            time.Time
		_preferredLocation   string
		_cronExpression      string
		_isSchedulePerSecond bool
		_timeLocation        *time.Location
		_secondInterval      *int
		_invocableType       *IInvocable
		_scheduler           *Scheduler
		_uniqueId            string
	}

	ScheduleEventConfig func(*ScheduleEvent)
)

func WithScheduleFunc[T any, R bool](predicate delegate.Func[T, R]) ScheduleEventConfig {
	return func(event *ScheduleEvent) {}
}

func WithScheduleAction[T any](predicate delegate.Action[T]) ScheduleEventConfig {
	return func(event *ScheduleEvent) {}
}

func NewScheduleEvent(configs ...ScheduleEventConfig) *ScheduleEvent {
	s := &ScheduleEvent{}
	for _, config := range configs {
		config(s)
	}

	s._zoneTime = time.Now().UTC()
	s._isSchedulePerSecond = s._cronExpression != ""

	if s._preferredLocation != "" {
		location, err := time.LoadLocation(s._preferredLocation)
		if err != nil {
			panic(err)
		}

		s._timeLocation = location
		s._zoneTime = time.Now().In(location)
	}

	return s
}

func (e *ScheduleEvent) InvokeScheduledEvent(ctx context.Context) {
	if e.WhenPredicateFails() {
		return
	}

	if e._invocableType == nil {
		go e._scheduleAction.Invoke()
	} else {
		invocableValue := reflect.ValueOf(e._invocableType)
		if invocableValue.Kind() == reflect.Ptr {
			invocable := invocableValue.Elem().Interface().(IInvocable)
			go func() {
				_, err := invocable.Invoke()
				if err != nil {
					panic(err)
				}
			}()

		}
	}

	e.markAsExecuteOnce()
	e.unscheduleIfWarranted()
}

func (e *ScheduleEvent) WhenPredicateFails() bool {
	return e._whenPredicate != nil && !e._whenPredicate()
}
func (e *ScheduleEvent) IsCronBasedTask() bool               { return !e._isSchedulePerSecond }
func (e *ScheduleEvent) OverlappingUniqueIdentifier() string { return e._uniqueId }
func (e *ScheduleEvent) markAsExecuteOnce()                  { e._wasPreviouslyRun = true }
func (e *ScheduleEvent) previouslyRanAndMarkedToRunOnlyOnce() bool {
	return e._runOnce && e._wasPreviouslyRun
}

func (e *ScheduleEvent) unscheduleIfWarranted() {
	if e._scheduler != nil && e.previouslyRanAndMarkedToRunOnlyOnce() {
		e._scheduler.TryUnschedule(e._uniqueId)
	}
}

func (e *ScheduleEvent) IsDue(now time.Time) bool {
	zoneTime := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		now.Nanosecond(),
		e._timeLocation,
	)

	if e._isSchedulePerSecond {
		if now.Second() == 0 {
			return ONE_MINUTE_AS_SECOND%*e._secondInterval == 0
		} else {
			return now.Second()%*e._secondInterval == 0
		}
	} else {

		if e._cronExpression == "" {
			panic(errors.New("no cron expression"))
		}

		g := gronx.New()
		due, err := g.IsDue(e._cronExpression, zoneTime)
		if err != nil {
			panic(err)
		}

		return due
	}
}
