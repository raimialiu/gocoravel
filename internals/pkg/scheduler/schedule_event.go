package scheduler

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/adhocore/gronx"
	"github.com/raimialiu/gocoravel/internals/pkg/system/delegate"
)

// DFG
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
		_secondInterval      int
		_invocableType       *IInvocable
		_scheduler           *Scheduler
		_uniqueId            string
	}

	ScheduleEventConfig func(*ScheduleEvent)
)

func (s *ScheduleEvent) NotSecondBase() {
	s._isSchedulePerSecond = false
}

func (e *ScheduleEvent) EveryFiveSeconds() *ScheduleEvent {
	secondValue := 5
	e._isSchedulePerSecond = true
	e._secondInterval = secondValue

	return e
}

func (e *ScheduleEvent) EverySecondAt(t int) *ScheduleEvent {
	e._isSchedulePerSecond = true
	e._secondInterval = t

	return e
}

func (e *ScheduleEvent) EverySecond() *ScheduleEvent {
	secondValue := 1
	e._isSchedulePerSecond = true
	e._secondInterval = secondValue

	return e
}

func (e *ScheduleEvent) Hourly() *ScheduleEvent {
	e._cronExpression = "00 * * * *"
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) Daily() *ScheduleEvent {
	e._cronExpression = "00 00 * * *"
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) DailyAt(t int) *ScheduleEvent {
	e._cronExpression = fmt.Sprintf("%d 00 * * *", t)
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) HourlyAt(t int) *ScheduleEvent {
	e._cronExpression = fmt.Sprintf("%d 00 * * * *", t)
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) EveryMinute() *ScheduleEvent {
	e._cronExpression = "* * * * *"
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) EveryFiveMinutes() *ScheduleEvent {
	e._cronExpression = "*/5 * * * *"
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) EveryTenMinutes() *ScheduleEvent {
	e._cronExpression = "*/10 * * * *"
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) EveryThirtyMinutes() *ScheduleEvent {
	e._cronExpression = "*/30 * * * *"
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) Weekly() *ScheduleEvent {
	e._cronExpression = "00 00 * * 1"
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) WeeklyAt(t int) *ScheduleEvent {
	e._cronExpression = fmt.Sprintf("00 00 * * %d", t)
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) Monthly() *ScheduleEvent {
	e._cronExpression = "00 00 1 * *"
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) MonthlyAt(t int) *ScheduleEvent {
	e._cronExpression = fmt.Sprintf("00 00 %d * *", t)
	e.NotSecondBase()
	return e
}

func (e *ScheduleEvent) Cron(expression string) *ScheduleEvent {
	e._cronExpression = expression
	e.NotSecondBase()
	return e
}

func WithScheduleFunc[T any, R bool](predicate delegate.Func[interface{}, interface{}]) ScheduleEventConfig {
	return func(event *ScheduleEvent) {
		event._scheduleAction = *delegate.NewAction(nil, predicate)
	}
}

func WithInvocableType(invocableType any) ScheduleEventConfig {
	return func(event *ScheduleEvent) {
		if vl, ok := invocableType.(IInvocable); ok {
			event._invocableType = &vl
		}
	}
}

func WithInvocableTypeAndParams(invocableType any, params ...interface{}) ScheduleEventConfig {
	return func(event *ScheduleEvent) {
		if vl, ok := invocableType.(IInvocable); ok {
			event._invocableType = &vl
			event._parameters = params
		}
	}
}

func WithScheduleAction[T any](predicate delegate.Action[interface{}]) ScheduleEventConfig {
	return func(event *ScheduleEvent) {
		event._scheduleAction = *delegate.NewAction(predicate, nil)
	}
}

func (e *ScheduleEvent) PreventOverlapping(name string) {
	e._preventOverlapping = true
	e.Name(name)
}

func NewScheduleEvent(configs ...ScheduleEventConfig) *ScheduleEvent {
	s := &ScheduleEvent{}
	for _, config := range configs {
		config(s)
	}

	if s._preferredLocation == "" {
		s._preferredLocation = "Africa/Lagos"
	}
	s._zoneTime = time.Now().UTC()
	s._isSchedulePerSecond = s._cronExpression == ""

	location, err := time.LoadLocation(s._preferredLocation)
	if err != nil {
		panic(err)
	}

	s._timeLocation = location
	s._zoneTime = time.Now().In(location)

	return s
}

func (e *ScheduleEvent) InvokeScheduledEvent(t time.Time) {
	if e.WhenPredicateFails() || !e.IsDue(t) {
		return
	}

	if e._invocableType == nil {
		go e._scheduleAction.Invoke()
	} else {
		invocableValue := reflect.ValueOf(e._invocableType)
		if invocableValue.Kind() == reflect.Ptr {
			if invocable, ok := invocableValue.Elem().Interface().(IInvocable); ok {
				_, err := e._Invoke(&invocable, e._parameters)
				if err != nil {
					return
				}
			}
		} else {
			_, err := e._Invoke(e._invocableType, e._parameters)
			if err != nil {
				return
			}
		}
	}

	e.markAsExecuteOnce()
	e.unscheduleIfWarranted()
}

func (e *ScheduleEvent) _Invoke(invocableType *IInvocable, params ...interface{}) (bool, error) {
	if invocableType == nil {
		panic(errors.New("invocable type is nil"))
	}
	invocable := *invocableType
	if len(e._parameters) == 0 {
		go func() {
			_, err := invocable.Invoke()
			if err != nil {
				panic(err)
			}
		}()
		return true, nil
	}

	go func() {
		_, err := invocable.InvokeWithPayload(params...)
		if err != nil {
			panic(err)
		}
	}()
	return true, nil
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

func (e *ScheduleEvent) Name(value string) *ScheduleEvent {
	e._uniqueId = value
	return e
}

func (e *ScheduleEvent) RunOnceAtStart() bool { return e._runOnce }

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
		if e._secondInterval <= 0 {
			return false
		}

		if now.Second() == 0 {
			return ONE_MINUTE_AS_SECOND%e._secondInterval == 0
		} else {
			return now.Second()%e._secondInterval == 0
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
