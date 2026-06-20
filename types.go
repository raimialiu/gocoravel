package main

import (
	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
	"github.com/raimialiu/gocoravel/internals/pkg/services"
)

type (
	Coravel struct {
		_hostedServices []services.HostedService
		_scheduler      scheduler.Scheduler
	}
)
