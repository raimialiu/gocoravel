package main

import "context"

type (
	Invocable interface {
		Invoke(ctx context.Context)
	}
	InvocableWithPayload[T any] interface {
		Invoke(ctx context.Context)
		Payload() T
	}
	Scheduler    interface{}
	ScheduleTask struct{}
)
