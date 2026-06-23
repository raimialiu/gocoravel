# gocoravel

A task scheduler for Go, inspired by [Laravel Coravel](https://docs.coravel.net/). Schedule work in code with a fluent API, optionally persist it to Redis so schedules and run history survive restarts, and watch/control everything from a live web dashboard.

- **Fluent scheduling** — `EverySecond()`, `EveryFiveSeconds()`, `EveryMinute()`, `Daily()`, `Cron("*/5 * * * *")`, …
- **Two kinds of jobs** — anonymous closures, or typed *invocables* (so they can be rebuilt on restart).
- **Optional persistence** — pluggable storage providers (Redis included); schedules + per-run history are saved, and missed runs are **replayed** on restart.
- **Live dashboard** — a Bull/Temporal-style UI: overview stats, a jobs list, a per-job detail page (metrics, status timeline, run history, errors) and controls (run, pause/resume, delete, clear history, replay). Updates live over Server-Sent Events. Mount it on your own server or run it standalone.
- **Error surfacing** — a single `OnError(func(error))` hook.
- **Few dependencies** — the scheduler and dashboard use only the standard library (`net/http`); the Redis provider uses [`go-redis/v9`](https://github.com/redis/go-redis).

---

## Table of contents

- [Requirements](#requirements)
- [Install](#install)
- [Quick start](#quick-start)
- [Core concepts](#core-concepts)
- [Scheduling API](#scheduling-api)
- [Jobs: closures vs invocables](#jobs-closures-vs-invocables)
- [Persistence](#persistence)
- [Replay & backfill](#replay--backfill)
- [Dashboard](#dashboard)
- [Error handling](#error-handling)
- [Full example](#full-example)
- [Architecture](#architecture)
- [Extending: custom storage providers](#extending-custom-storage-providers)
- [Notes & limitations](#notes--limitations)
- [Roadmap](#roadmap)

---

## Requirements

- **Go 1.26+** (uses generics and the 1.22 `net/http` method/pattern router).
- **Redis** — *optional*, only needed if you enable persistence.

## Install

```bash
go get github.com/raimialiu/gocoravel
```

Packages live under `internals/pkg/…`:

| Package | Import path |
| --- | --- |
| Facade | `github.com/raimialiu/gocoravel/internals/pkg/coravel` |
| Scheduler | `github.com/raimialiu/gocoravel/internals/pkg/scheduler` |
| Dashboard | `github.com/raimialiu/gocoravel/internals/pkg/dashboard` |
| Storage providers | `github.com/raimialiu/gocoravel/internals/pkg/store/providers` |

## Quick start

No Redis required — this runs the scheduler in-memory:

```go
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/raimialiu/gocoravel/internals/pkg/coravel"
)

func main() {
	c := coravel.NewCoravel().AddScheduler()

	c.UseScheduler(func(cv *coravel.Coravel) {
		s := cv.Scheduler()

		s.ScheduleSimple(func() {
			println("every second")
		}).EverySecond().Name("tick")

		s.ScheduleSimple(func() {
			println("top of every minute")
		}).EveryMinute().Name("minutely")
	})

	// block until Ctrl+C
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	c.Stop()
}
```

---

## Core concepts

`Coravel` is the facade. You build it with a chain, then register jobs:

```go
c := coravel.NewCoravel().
	OnError(func(err error) { log.Printf("coravel: %v", err) }).   // optional
	AddPersistenceStorage(cfg, providers.Redis).                   // optional
	AddScheduler().                                                // required
	AddDashboard(dashboard.Options{Addr: ":8099"})                 // optional
```

**Order matters:**

1. `OnError` and `AddPersistenceStorage` must come **before** `AddScheduler` (the scheduler captures them as it starts).
2. `AddDashboard` comes **after** `AddScheduler` (it binds to the running scheduler).
3. `Replay()` (if you use persistence) is called **after** you register your jobs.

Jobs are registered inside `UseScheduler` via `cv.Scheduler()`, which returns the `*scheduler.Scheduler`.

---

## Scheduling API

Every `Schedule*` method returns a `*scheduler.ScheduleEvent` you can chain a cadence and modifiers onto.

### Register a job

| Method | Use it for |
| --- | --- |
| `ScheduleSimple(func())` | a plain closure with no return |
| `ScheduleFunc(func(payload ...interface{}) interface{})` | a closure that returns a value |
| `SchedulePureFunc(func(params ...any) any, params ...any)` | a closure + bound params |
| `ScheduleInvocable(v any)` | a value implementing `IInvocable` (recommended for persistence) |
| `ScheduleInvocableWithParams(v any, params ...interface{})` | an invocable + params |

### Cadence

**Sub-minute** (fires when `clock-second % n == 0`):

| Method | Meaning |
| --- | --- |
| `EverySecond()` | every second |
| `EveryFiveSeconds()` | every 5 seconds |
| `EverySecondAt(n)` | every `n` seconds |

**Cron-based** (evaluated at the top of each minute):

| Method | Cron |
| --- | --- |
| `EveryMinute()` | `* * * * *` |
| `EveryFiveMinutes()` | `*/5 * * * *` |
| `EveryTenMinutes()` | `*/10 * * * *` |
| `EveryThirtyMinutes()` | `*/30 * * * *` |
| `Hourly()` / `HourlyAt(m)` | `00 * * * *` / `m 00 * * *` |
| `Daily()` / `DailyAt(h)` | `00 00 * * *` / `h 00 * * *` |
| `Weekly()` / `WeeklyAt(d)` | `00 00 * * 1` / `00 00 * * d` |
| `Monthly()` / `MonthlyAt(d)` | `00 00 1 * *` / `00 00 d * *` |
| `Cron(expr)` | any 5-field cron expression (via [`adhocore/gronx`](https://github.com/adhocore/gronx)) |

### Modifiers

| Method | Effect |
| --- | --- |
| `Name(id string) *ScheduleEvent` | sets the job's stable id (chainable). **Name your jobs** — it's the key used for persistence, control, and the dashboard. |
| `PreventOverlapping(name string)` | marks the job non-overlapping and names it |
| `EnsurePersistence()` | opt this job into persistence (schedule + run history) |

> `PreventOverlapping` and `EnsurePersistence` return nothing — call them **last**, or on their own line:
> ```go
> j := s.ScheduleSimple(fn).EveryFiveSeconds().Name("report")
> j.EnsurePersistence()
> ```

---

## Jobs: closures vs invocables

A Go function **cannot be serialized**, so the two job styles behave differently across restarts:

- **Closures** (`ScheduleSimple` / `ScheduleFunc`) — your app must re-declare them on every boot. Persistence restores their *state and history*; the closure itself comes from your code.
- **Invocables** (`ScheduleInvocable`) — a named type implementing `IInvocable`. If you register a constructor (see below), coravel can **rebuild and reschedule it on restart even if you don't re-declare it**.

```go
type IInvocable interface {
	Invoke() (bool, error)
	InvokeWithPayload(payload ...interface{}) (bool, error)
}
```

```go
type ReportJob struct{}

func (ReportJob) Invoke() (bool, error)                          { /* … */ return true, nil }
func (ReportJob) InvokeWithPayload(p ...interface{}) (bool, error) { return ReportJob{}.Invoke() }

// register a constructor so it can be rebuilt from storage on restart
scheduler.RegisterInvocable(&ReportJob{}, func() scheduler.IInvocable { return &ReportJob{} })

// schedule it
s.ScheduleInvocable(&ReportJob{}).Hourly().Name("report")
```

A job that returns an `error` (or panics) is recorded as **failed** with the message captured — visible in the dashboard's Errors view and surfaced through `OnError`.

---

## Persistence

Enable persistence by adding a storage backend. A single `ConnectionConfiguration` cascades to every provider you list:

```go
import "github.com/raimialiu/gocoravel/internals/pkg/store/providers"

c := coravel.NewCoravel().
	AddPersistenceStorage(
		providers.ConnectionConfiguration{Host: "127.0.0.1", Port: 6379},
		providers.Redis,
	).
	AddScheduler()
```

`ConnectionConfiguration`:

| Field | Notes |
| --- | --- |
| `Url` | full connection URL, e.g. `redis://user:pass@host:6379/0`. If set, it wins over `Host`/`Port`. |
| `Host`, `Port` | default `127.0.0.1:6379` |
| `Username`, `Password` | optional auth |
| `PoolSize` | optional connection pool size |

Persistence is **opt-in per job** via `EnsurePersistence()`. Persisted data (Redis keys):

| Key | Contents |
| --- | --- |
| `coravel:schedules` | list of persisted job ids |
| `coravel:schedule:<id>` | the job's `JobSchedule` (JSON) |
| `coravel:runs:<id>` | the job's run history (list of `JobRun` JSON) |

## Replay & backfill

When persistence is on, call `Replay()` once **after** registering your jobs (or `coravel` does it for you when you call `c.Replay()`):

```go
c.UseScheduler(func(cv *coravel.Coravel) { /* register jobs */ })
if err := c.Replay(); err != nil {
	log.Printf("replay: %v", err)
}
```

`Replay()`:

1. persists the current jobs' schedules,
2. rebuilds any **registered invocables** that weren't re-declared and reschedules them,
3. restores run-once state (a one-shot job that already ran won't run again),
4. **backfills** the runs each job missed while the process was down, oldest-first.

Backfill is bounded so it can't stampede: it scans at most `100_000` steps (keeping the most recent window) and replays at most `500` occurrences per job; truncation is reported through `OnError`.

---

## Dashboard

A live web UI to visualize and control the scheduler.

```go
import "github.com/raimialiu/gocoravel/internals/pkg/dashboard"

// (A) managed standalone server — started in the background, stopped by c.Stop()
c.AddDashboard(dashboard.Options{Addr: ":8099", BasePath: "/coravel"})
// open http://localhost:8099/coravel/

// (B) mount on your own server
mux := http.NewServeMux()
mux.Handle("/coravel/", c.Dashboard(dashboard.Options{BasePath: "/coravel"}))
http.ListenAndServe(":8080", mux)
```

`dashboard.Options`:

| Field | Default | Notes |
| --- | --- | --- |
| `Addr` | `:8099` | listen address (for `AddDashboard`) |
| `BasePath` | `/coravel` | path the UI + API mount under |
| `Title` | `Coravel` | shown in the UI |
| `Token` | *(none)* | if set, requests must carry `?token=…` or `Authorization: Bearer …` |

> When mounting yourself, the routes already include `BasePath` — mount with `mux.Handle(BasePath+"/", …)` and **do not** `StripPrefix`.

**Views:** Overview (global stats + recent activity), Jobs (table), Job detail (metric cards, a success/failure timeline, full run history, per-job actions), and Errors (recent failures across all jobs).

**HTTP API** (under `BasePath`):

| Method & path | Action |
| --- | --- |
| `GET /api/overview` | global stats + recent activity |
| `GET /api/jobs` | all jobs |
| `GET /api/jobs/{id}/runs` | a job's run history |
| `GET /api/jobs/{id}/metrics` | a job's aggregated metrics |
| `GET /api/errors` | recent failed runs |
| `GET /api/stream` | Server-Sent Events (live updates) |
| `POST /api/jobs/{id}/trigger` | run now |
| `POST /api/jobs/{id}/pause` / `…/resume` | pause / resume |
| `POST /api/replay` | replay (persistence only) |
| `DELETE /api/jobs/{id}` | unschedule |
| `DELETE /api/jobs/{id}/runs` | clear history |

Run history and metrics are kept in memory too, so the dashboard works **even without Redis**.

---

## Error handling

Register one handler for non-fatal errors — failed jobs, persistence write failures, dashboard startup errors, backfill truncation:

```go
coravel.NewCoravel().OnError(func(err error) {
	log.Printf("coravel: %v", err)
})
```

If no handler is set, such errors are dropped (dashboard server errors fall back to the standard logger).

---

## Full example

```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/raimialiu/gocoravel/internals/pkg/coravel"
	"github.com/raimialiu/gocoravel/internals/pkg/dashboard"
	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
	"github.com/raimialiu/gocoravel/internals/pkg/store/providers"
)

type GreetJob struct{ Who string }

func (g *GreetJob) Invoke() (bool, error) {
	fmt.Println("hello,", g.Who)
	return true, nil
}
func (g *GreetJob) InvokeWithPayload(p ...interface{}) (bool, error) { return g.Invoke() }

func main() {
	c := coravel.NewCoravel().
		OnError(func(err error) { log.Printf("coravel: %v", err) }).
		AddPersistenceStorage(
			providers.ConnectionConfiguration{Host: "127.0.0.1", Port: 6379},
			providers.Redis,
		).
		AddScheduler().
		AddDashboard(dashboard.Options{Addr: ":8099", BasePath: "/coravel"})

	// invocables registered here can be rebuilt on restart
	scheduler.RegisterInvocable(&GreetJob{}, func() scheduler.IInvocable { return &GreetJob{} })

	c.UseScheduler(func(cv *coravel.Coravel) {
		s := cv.Scheduler()

		s.ScheduleSimple(func() { fmt.Println("tick") }).EverySecond().Name("tick")

		report := s.ScheduleInvocable(&GreetJob{Who: "world"}).EveryFiveSeconds().Name("greet")
		report.EnsurePersistence()
	})

	if err := c.Replay(); err != nil { // reconcile + backfill on boot
		log.Printf("replay: %v", err)
	}

	fmt.Println("dashboard: http://localhost:8099/coravel/")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	c.Stop()
}
```

A runnable version of this lives in [`main.go`](./main.go) (toggle `usePersistence`).

---

## Architecture

```
                 ┌──────────────────────────── coravel.Coravel ───────────────────────────┐
                 │  builder: OnError · AddPersistenceStorage · AddScheduler · AddDashboard  │
                 └───────┬───────────────────────┬───────────────────────────┬─────────────┘
                         │                        │                           │
                 scheduler.Scheduler      store.CoravelStore           dashboard.Handler
                 ├ _tasks (jobs)           └ DataSource (router) ──┐    (http.Handler: UI + JSON API + SSE)
                 ├ feed (runs/metrics/SSE)                         │
                 └ _store ─────────────────────────────────────►  providers.Provider
                                                                   └ RedisDataProvider (go-redis v9)
```

- **`scheduler`** owns the jobs (`ScheduleEvent`), the run loop, and an in-memory **feed** (recent runs + cumulative metrics + an SSE event hub).
- **`store` → `DataSource` → `providers.Provider`** is the persistence layer. `DataSource` routes to the active provider by its `Kind()` (interface-driven, no type switch), so new backends drop in via a registry.
- **`dashboard`** is a thin `net/http` layer over the scheduler's read/control methods, with an embedded single-page UI.

## Extending: custom storage providers

Implement `providers.Provider` (which is `DataProvider` + `DataConnector` + `Kind()`), register a constructor, then select it by kind:

```go
providers.RegisterProvider("postgres", func(cfg providers.ConnectionConfiguration) providers.Provider {
	return NewPostgresProvider(cfg)
})

c.AddPersistenceStorage(cfg, providers.ProviderKind("postgres"))
```

`DataProvider` is a small key/value + list CRUD contract (`Get`, `GetSingle`, `Create`, `CreateMany`, `CreateIfNotExists`, `Update`, `UpdateMany`, `Delete`).

## Notes & limitations

- **Name your persistent jobs.** The id (`Name`) is the storage/control key. Unnamed jobs still run, but a stable name is what lets replay reconnect them across restarts.
- **Cron resolution is one minute**; sub-minute cadence uses the second-based helpers.
- **Backfill replays every missed tick** (bounded). For high-frequency jobs after a long outage that can be a burst on boot; the bounds cap it, and the policy is easy to change to "catch up once."
- The in-memory run history shown on the dashboard is a capped recent buffer per job; full history lives in the store when persistence is on.

## Roadmap

- **V1** — Scheduler ✅ · Events · Queue · Message distributor · Redis store ✅ · Dashboard ✅
- **V2** — multiple configurable stores.

---

*Inspired by Laravel Coravel. Built in Go.*
