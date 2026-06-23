package coravel

import (
	"context"
	"net/http"

	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
	"github.com/raimialiu/gocoravel/internals/pkg/store"
)

type (
	Coravel struct {
		_host               scheduler.SchedulerHost
		_scheduler          scheduler.Scheduler
		_ctx                context.Context
		_dataStore          *store.CoravelStore
		_persistenceEnabled bool
		_errorHandler       func(error)
		_dashboard          *http.Server
	}
)
