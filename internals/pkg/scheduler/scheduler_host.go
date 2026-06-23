package scheduler

import (
	"context"
	"sync"
	"time"
)

type SchedulerHost struct {
	_scheduler    *Scheduler
	_timer        time.Ticker
	_ctx          context.Context
	_cancel       context.CancelFunc
	_previousTick time.Time
}

func NewSchedulerHost(scheduler *Scheduler) *SchedulerHost {
	t := time.Now()
	return &SchedulerHost{
		_previousTick: t,
		_scheduler:    scheduler,
	}
}
func (s *SchedulerHost) Start() {
	now := time.Now()
	var lck sync.Mutex

	// lock
	lck.Lock()
	ticks := s.GetTicksBetweenPreviousAndNext(now)
	s.setNextTick(now)
	lck.Unlock()

	if len(ticks) > 0 {
		for _, t := range ticks {
			go s.RunSchedulerPerSecond(t)
		}
	}

	go s.RunSchedulerPerSecond(now)
}

func (s *SchedulerHost) RunSchedulerPerSecond(t time.Time) {
	ticker := time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case <-s._ctx.Done():
				return
			case <-ticker.C:
				go s._scheduler.RunAt(t)
			}
		}
	}()
}

func (s *SchedulerHost) preciseSeconds(t time.Time) time.Time {
	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		t.Second(),
		t.Nanosecond(),
		t.Location(),
	).Add(1 * time.Second)
}

func (s *SchedulerHost) setNextTick(t time.Time) {
	s._previousTick = t
}

func (s *SchedulerHost) GetTicksBetweenPreviousAndNext(t time.Time) []time.Time {
	ticks := make([]time.Time, 0)
	nextTick := s.preciseSeconds(s._previousTick).Add(1 * time.Second)

	for nextTick.Before(t) {
		ticks = append(ticks, nextTick)
		nextTick = s.preciseSeconds(nextTick).Add(1 * time.Second)
	}

	return ticks
}
