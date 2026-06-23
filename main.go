package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/raimialiu/gocoravel/internals/pkg/coravel"
	"github.com/raimialiu/gocoravel/internals/pkg/dashboard"
	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
	"github.com/raimialiu/gocoravel/internals/pkg/store/providers"
)

// usePersistence enables Redis-backed persistence + replay.
// Flip to true once a Redis server is running on 127.0.0.1:6379.
const usePersistence = true

// GreetJob is an example IInvocable: a job declared as a type instead of a
// closure. Type-based jobs can be rebuilt from storage and rescheduled on
// restart (see scheduler.RegisterInvocable below); closures cannot, so the app
// must re-declare them each boot.
type GreetJob struct {
	Who string
}

func (g *GreetJob) Invoke() (bool, error) {
	fmt.Printf("[%s] greet invocable: hello, %s\n", stamp(), g.Who)
	return true, nil
}

func (g *GreetJob) InvokeWithPayload(payload ...interface{}) (bool, error) {
	return g.Invoke()
}

// FlakyJob fails on every other run, to demonstrate failed runs / the Errors view.
type FlakyJob struct{ runs atomic.Int64 }

func (f *FlakyJob) Invoke() (bool, error) {
	n := f.runs.Add(1)
	if n%2 == 0 {
		return false, fmt.Errorf("simulated failure on run #%d", n)
	}
	fmt.Printf("[%s] flaky ok (run #%d)\n", stamp(), n)
	return true, nil
}

func (f *FlakyJob) InvokeWithPayload(payload ...interface{}) (bool, error) {
	return f.Invoke()
}

func main() {
	// 1) Build coravel. OnError and AddPersistenceStorage must come BEFORE
	//    AddScheduler — the scheduler picks them up as it starts.
	builder := coravel.NewCoravel().
		OnError(func(err error) { log.Printf("coravel: %v", err) })

	if usePersistence {
		// One ConnectionConfiguration cascades to every provider you list.
		builder = builder.AddPersistenceStorage(
			// DB 15 keeps coravel's keys isolated from anything else on this Redis.
			providers.ConnectionConfiguration{Url: "redis://127.0.0.1:6379/15"},
			providers.Redis,
		)
		// Let type-based jobs be rebuilt on restart.
		scheduler.RegisterInvocable(&GreetJob{}, func() scheduler.IInvocable { return &GreetJob{} })
		scheduler.RegisterInvocable(&FlakyJob{}, func() scheduler.IInvocable { return &FlakyJob{} })
	}

	c := builder.
		AddScheduler().
		AddDashboard(dashboard.Options{Addr: ":8099", BasePath: "/coravel"})

	// 2) Register jobs. Heads-up: EnsurePersistence() and PreventOverlapping()
	//    return nothing, so call them LAST (or on their own line, as below).
	c.UseScheduler(func(cv *coravel.Coravel) {
		s := cv.Scheduler()

		// a) closure, every second
		s.ScheduleSimple(func() {
			fmt.Printf("[%s] tick — every second\n", stamp())
		}).EverySecond().Name("tick")

		// b) closure, every 5s, with run history persisted (when persistence is on)
		five := s.ScheduleSimple(func() {
			fmt.Printf("[%s] every five seconds\n", stamp())
		}).EveryFiveSeconds().Name("five-seconds")
		five.EnsurePersistence()

		// c) type-based (IInvocable) job, every 5s, persisted
		greet := s.ScheduleInvocable(&GreetJob{Who: "world"}).EveryFiveSeconds().Name("greet")
		greet.EnsurePersistence()

		// c2) a job that fails on every other run — appears in Errors + failed metrics
		flaky := s.ScheduleInvocable(&FlakyJob{}).EveryFiveSeconds().Name("flaky")
		flaky.EnsurePersistence()

		// d) cron job — fires at the top of every minute
		s.ScheduleSimple(func() {
			fmt.Printf("[%s] top of the minute — cron\n", stamp())
		}).EveryMinute().Name("minutely")
	})

	// 3) With persistence on, reconcile saved state and backfill any runs missed
	//    while the app was down. Call AFTER registering jobs.
	if usePersistence {
		if err := c.Replay(); err != nil {
			log.Printf("replay: %v", err)
		}
	}

	fmt.Printf("coravel running (persistence=%v) — dashboard: http://localhost:8099/coravel/ — Ctrl+C to stop\n", usePersistence)

	// 4) Block until interrupted, then stop the scheduler.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	c.Stop()
	fmt.Println("stopped")
}

func stamp() string { return time.Now().Format("15:04:05") }

// Roadmap — V1: Scheduler, Events, Queue, Message Distributor, Redis store.
//           V2: multiple configurable stores.
