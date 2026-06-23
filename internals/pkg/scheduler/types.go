package scheduler

import "github.com/raimialiu/gocoravel/internals/pkg/system/delegate"

type (
	IScheduler interface {
		Schedule(actionToSchedule delegate.Action[interface{}]) *ScheduleEvent
	}

	IScheduleInterval interface {
		Hourly() *ScheduleEvent
		Daily() *ScheduleEvent
		DailyAt(t int) *ScheduleEvent
		HourlyAt(t int) *ScheduleEvent
		EveryMinute() *ScheduleEvent
		EveryFiveMinutes() *ScheduleEvent
		EveryTenMinutes() *ScheduleEvent
		EveryThirtyMinutes() *ScheduleEvent
		Weekly() *ScheduleEvent
		WeeklyAt(t int) *ScheduleEvent
		Monthly() *ScheduleEvent
		MonthlyAt(t int) *ScheduleEvent
		Cron(expression string) *ScheduleEvent
		EverySecond() *ScheduleEvent
		EveryFiveSeconds() *ScheduleEvent
		EverySecondAt(t int) *ScheduleEvent
	}

	IInvocable interface {
		Invoke() (bool, error)
		InvokeWithPayload(payload ...interface{}) (bool, error)
	} // only this is executed (by calling the invoke methods)

	InvocableExecutor struct{} // the part that runs the job

	ScheduleTask struct {
	} // structure of a job
)
