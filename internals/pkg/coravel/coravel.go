package coravel

import (
	"context"
	"log"
	"net/http"

	"github.com/raimialiu/gocoravel/internals/pkg/dashboard"
	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
	"github.com/raimialiu/gocoravel/internals/pkg/store"
	"github.com/raimialiu/gocoravel/internals/pkg/store/providers"
)

type CoravelConfig func(scheduler *Coravel)

func NewCoravel(configs ...CoravelConfig) *Coravel {
	coravel := &Coravel{}

	for _, config := range configs {
		config(coravel)
	}

	return coravel
}

// OnError registers a handler for non-fatal errors surfaced by the scheduler
// (e.g. a persistence write that fails). Call it before AddScheduler.
func (c *Coravel) OnError(handler func(error)) *Coravel {
	c._errorHandler = handler
	return c
}

// AddPersistenceStorage enables persistence. The single ConnectionConfiguration
// cascades into every requested provider: each kind is built from cfg via the
// providers factory, registered in a DataSource keyed by Kind, and the active
// one is opened.
//
// Call this before AddScheduler so the scheduler picks up the store.
func (c *Coravel) AddPersistenceStorage(cfg providers.ConnectionConfiguration, active providers.ProviderKind, extra ...providers.ProviderKind) *Coravel {
	kinds := append([]providers.ProviderKind{active}, extra...)

	provs := make([]providers.Provider, 0, len(kinds))
	for _, kind := range kinds {
		p, err := providers.New(kind, cfg) // cfg cascades into each provider
		if err != nil {
			panic(err)
		}
		provs = append(provs, p)
	}

	ds, err := store.NewDataSource(active, provs...)
	if err != nil {
		panic(err)
	}
	ds.Open()

	c._dataStore = store.NewCoravelStore(ds)
	c._persistenceEnabled = true
	return c
}

func (c *Coravel) AddScheduler() *Coravel {
	ctx, cancel := context.WithCancel(context.Background())
	s := scheduler.NewScheduler(ctx, cancel)
	if c._dataStore != nil {
		s.UseStore(c._dataStore)
	}
	if c._errorHandler != nil {
		s.OnError(c._errorHandler)
	}
	h := scheduler.NewSchedulerHost(s, ctx, cancel)

	go h.Start()
	c._scheduler = *s
	c._host = *h
	c._ctx = ctx
	return c
}

func (c *Coravel) Scheduler() *scheduler.Scheduler {
	return &c._scheduler
}

func (c *Coravel) Stop() {
	c._scheduler.CancelTasks()
	if c._dashboard != nil {
		_ = c._dashboard.Close()
	}
}

func (c *Coravel) UseScheduler(configs ...CoravelConfig) {
	for _, config := range configs {
		config(c)
	}
}

// Replay reconciles persisted state with the registered jobs: it persists their
// schedules, rebuilds any registered invocables that weren't re-declared,
// restores run-once state, and catches up jobs that never ran. It is a no-op
// unless persistence was enabled via AddPersistenceStorage. Call it after
// registering your jobs.
func (c *Coravel) Replay() error {
	if !c._persistenceEnabled {
		return nil
	}
	return c._scheduler.Replay()
}

// Dashboard returns a mountable http.Handler serving the scheduler dashboard
// (UI + JSON API + live SSE) under opts.BasePath. Mount it on your own server:
//
//	mux.Handle("/coravel/", c.Dashboard(dashboard.Options{BasePath: "/coravel"}))
func (c *Coravel) Dashboard(opts dashboard.Options) http.Handler {
	return dashboard.Handler(&c._scheduler, opts)
}

// StartDashboard runs the dashboard on its own HTTP server at addr (blocking).
// Prefer AddDashboard for the managed, chainable form.
func (c *Coravel) StartDashboard(addr string, opts dashboard.Options) error {
	return http.ListenAndServe(addr, c.Dashboard(opts))
}

// AddDashboard starts the dashboard on its own HTTP server (in the background)
// and returns the Coravel for chaining. Call it after AddScheduler. The server
// is shut down by Stop, and a startup error is delivered to the OnError handler
// (or logged if none is set).
//
//	coravel.NewCoravel().AddScheduler().
//	    AddDashboard(dashboard.Options{Addr: ":8099", BasePath: "/coravel"})
func (c *Coravel) AddDashboard(opts dashboard.Options) *Coravel {
	addr := opts.Addr
	if addr == "" {
		addr = ":8099"
	}
	srv := &http.Server{Addr: addr, Handler: c.Dashboard(opts)}
	c._dashboard = srv

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if c._errorHandler != nil {
				c._errorHandler(err)
			} else {
				log.Printf("coravel: dashboard: %v", err)
			}
		}
	}()
	return c
}
