package scheduler

import "github.com/raimialiu/gocoravel/internals/pkg/system/delegate"

type (
	IScheduler interface {
		Schedule(actionToSchedule delegate.Action[interface{}]) IScheduleInterval
	}

	IScheduleInterval interface {
	}

	IInvocable interface {
		Invoke() (bool, error)
		InvokeWithPayload(payload interface{}) (bool, error)
	} // only this is executed (by calling the invoke methods)

	InvocableExecutor struct{} // the part that runs the job

	ScheduleTask struct{} // structure of a job
)
