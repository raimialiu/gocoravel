package scheduler

import (
	"fmt"
	"sync"
	"time"
)

// JobView is a dashboard-facing snapshot of a scheduled job.
type JobView struct {
	Id         string    `json:"id"`
	Schedule   string    `json:"schedule"`
	Type       string    `json:"type"` // "closure" | "invocable"
	Persistent bool      `json:"persistent"`
	Paused     bool      `json:"paused"`
	RunOnce    bool      `json:"runOnce"`
	LastRun    time.Time `json:"lastRun"`
	LastStatus RunStatus `json:"lastStatus"`
}

// JobMetrics is the aggregated run statistics for one job.
type JobMetrics struct {
	Id          string    `json:"id"`
	Total       int       `json:"total"`
	Succeeded   int       `json:"succeeded"`
	Failed      int       `json:"failed"`
	SuccessRate float64   `json:"successRate"` // 0..1
	AvgMs       int64     `json:"avgMs"`
	LastRun     time.Time `json:"lastRun"`
	LastStatus  RunStatus `json:"lastStatus"`
}

// Overview is the global dashboard summary.
type Overview struct {
	Jobs      int      `json:"jobs"`
	Active    int      `json:"active"`
	Paused    int      `json:"paused"`
	TotalRuns int      `json:"totalRuns"`
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Recent    []JobRun `json:"recent"`
}

// DashboardEvent is pushed to dashboard subscribers over SSE.
type DashboardEvent struct {
	Type string    `json:"type"` // "snapshot" | "run"
	Jobs []JobView `json:"jobs,omitempty"`
	Run  *JobRun   `json:"run,omitempty"`
}

const (
	recentRunsPerJob = 100
	globalRecentRuns = 200
)

type jobStats struct {
	total      int
	succeeded  int
	failed     int
	sumMs      int64
	lastRun    time.Time
	lastStatus RunStatus
}

// feed keeps recent run history + cumulative stats in memory and fans events out
// to subscribers. It is always active (independent of persistence) so the
// dashboard is useful even without a store. A *feed is shared across the
// scheduler's value copies, so events from the running loop reach handlers built
// from a copy.
type feed struct {
	mu          sync.Mutex
	runs        map[string][]JobRun  // per-job recent runs (capped)
	stats       map[string]*jobStats // per-job cumulative counters
	recent      []JobRun             // global recent runs across all jobs (capped)
	subscribers map[int]chan DashboardEvent
	nextSub     int
}

func newFeed() *feed {
	return &feed{
		runs:        make(map[string][]JobRun),
		stats:       make(map[string]*jobStats),
		subscribers: make(map[int]chan DashboardEvent),
	}
}

func (f *feed) record(run JobRun) {
	f.mu.Lock()
	rs := append(f.runs[run.JobId], run)
	if len(rs) > recentRunsPerJob {
		rs = rs[len(rs)-recentRunsPerJob:]
	}
	f.runs[run.JobId] = rs

	st := f.stats[run.JobId]
	if st == nil {
		st = &jobStats{}
		f.stats[run.JobId] = st
	}
	st.total++
	switch run.Status {
	case RunSucceeded:
		st.succeeded++
	case RunFailed:
		st.failed++
	}
	st.sumMs += run.FinishedAt.Sub(run.StartedAt).Milliseconds()
	st.lastRun = run.FinishedAt
	st.lastStatus = run.Status

	f.recent = append(f.recent, run)
	if len(f.recent) > globalRecentRuns {
		f.recent = f.recent[len(f.recent)-globalRecentRuns:]
	}
	f.mu.Unlock()

	r := run
	f.publish(DashboardEvent{Type: "run", Run: &r})
}

func (f *feed) history(jobID string) []JobRun {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]JobRun, len(f.runs[jobID]))
	copy(out, f.runs[jobID])
	return out
}

func (f *feed) metrics(jobID string) JobMetrics {
	f.mu.Lock()
	defer f.mu.Unlock()
	m := JobMetrics{Id: jobID}
	if st := f.stats[jobID]; st != nil {
		m.Total = st.total
		m.Succeeded = st.succeeded
		m.Failed = st.failed
		m.LastRun = st.lastRun
		m.LastStatus = st.lastStatus
		if st.total > 0 {
			m.SuccessRate = float64(st.succeeded) / float64(st.total)
			m.AvgMs = st.sumMs / int64(st.total)
		}
	}
	return m
}

func (f *feed) totals() (total, succeeded, failed int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, st := range f.stats {
		total += st.total
		succeeded += st.succeeded
		failed += st.failed
	}
	return
}

func (f *feed) recentN(limit int, failedOnly bool) []JobRun {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]JobRun, 0, limit)
	for i := len(f.recent) - 1; i >= 0 && len(out) < limit; i-- {
		if failedOnly && f.recent[i].Status != RunFailed {
			continue
		}
		out = append(out, f.recent[i])
	}
	return out
}

func (f *feed) clear(jobID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.runs, jobID)
	delete(f.stats, jobID)
	kept := f.recent[:0]
	for _, r := range f.recent {
		if r.JobId != jobID {
			kept = append(kept, r)
		}
	}
	f.recent = kept
}

func (f *feed) subscribe() (int, chan DashboardEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextSub
	f.nextSub++
	ch := make(chan DashboardEvent, 16)
	f.subscribers[id] = ch
	return id, ch
}

func (f *feed) unsubscribe(id int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if ch, ok := f.subscribers[id]; ok {
		close(ch)
		delete(f.subscribers, id)
	}
}

func (f *feed) publish(ev DashboardEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ch := range f.subscribers {
		select {
		case ch <- ev:
		default: // drop for a slow consumer rather than block the scheduler
		}
	}
}

// -----------------------------------------------------------------------------
// Scheduler dashboard API (read + control + event stream)
// -----------------------------------------------------------------------------

// Jobs returns a snapshot of every registered job.
func (s *Scheduler) Jobs() []JobView {
	views := make([]JobView, 0)
	for _, job := range s._tasks.ToMap() {
		if job == nil {
			continue
		}
		view := JobView{
			Id:         job.OverlappingUniqueIdentifier(),
			Schedule:   job.describeSchedule(),
			Type:       job.jobType(),
			Persistent: job.ShouldPersist(),
			Paused:     job.IsPaused(),
			RunOnce:    job._runOnce,
		}
		if s._feed != nil {
			m := s._feed.metrics(view.Id)
			view.LastRun = m.LastRun
			view.LastStatus = m.LastStatus
		}
		views = append(views, view)
	}
	return views
}

// History returns the recent in-memory run history for a job (most recent last).
func (s *Scheduler) History(id string) []JobRun {
	if s._feed == nil {
		return nil
	}
	return s._feed.history(id)
}

// Metrics returns aggregated run statistics for a job.
func (s *Scheduler) Metrics(id string) JobMetrics {
	if s._feed == nil {
		return JobMetrics{Id: id}
	}
	return s._feed.metrics(id)
}

// Errors returns the most recent failed runs across all jobs.
func (s *Scheduler) Errors(limit int) []JobRun {
	if s._feed == nil {
		return nil
	}
	return s._feed.recentN(limit, true)
}

// Overview returns the global dashboard summary.
func (s *Scheduler) Overview() Overview {
	ov := Overview{}
	for _, job := range s._tasks.ToMap() {
		if job == nil {
			continue
		}
		ov.Jobs++
		if job.IsPaused() {
			ov.Paused++
		} else {
			ov.Active++
		}
	}
	if s._feed != nil {
		ov.TotalRuns, ov.Succeeded, ov.Failed = s._feed.totals()
		ov.Recent = s._feed.recentN(15, false)
	}
	return ov
}

// ClearHistory removes a job's recorded runs (in-memory and, if persistence is
// on, the stored run list).
func (s *Scheduler) ClearHistory(id string) {
	if s._feed != nil {
		s._feed.clear(id)
	}
	if s._store != nil {
		if _, err := s._store.DataSource.Delete(runsKey(id)); err != nil {
			s.reportError(err)
		}
	}
	s.publishSnapshot()
}

// Snapshot is the full current state, as a "snapshot" event.
func (s *Scheduler) Snapshot() DashboardEvent {
	return DashboardEvent{Type: "snapshot", Jobs: s.Jobs()}
}

// Subscribe returns a channel of dashboard events and a cancel func to release it.
func (s *Scheduler) Subscribe() (<-chan DashboardEvent, func()) {
	if s._feed == nil {
		s._feed = newFeed()
	}
	id, ch := s._feed.subscribe()
	return ch, func() { s._feed.unsubscribe(id) }
}

func (s *Scheduler) publishSnapshot() {
	if s._feed != nil {
		s._feed.publish(s.Snapshot())
	}
}

// TriggerNow runs a job immediately, out of schedule.
func (s *Scheduler) TriggerNow(id string) error {
	job := s.lookup(id)
	if job == nil {
		return fmt.Errorf("scheduler: no job %q", id)
	}
	go s.execute(job, time.Now())
	return nil
}

// Pause stops a job from firing until Resume is called.
func (s *Scheduler) Pause(id string) error {
	job := s.lookup(id)
	if job == nil {
		return fmt.Errorf("scheduler: no job %q", id)
	}
	job.setPaused(true)
	s.publishSnapshot()
	return nil
}

// Resume re-enables a paused job.
func (s *Scheduler) Resume(id string) error {
	job := s.lookup(id)
	if job == nil {
		return fmt.Errorf("scheduler: no job %q", id)
	}
	job.setPaused(false)
	s.publishSnapshot()
	return nil
}

// Unschedule removes a job from the scheduler by its id.
func (s *Scheduler) Unschedule(id string) bool {
	for key, job := range s._tasks.ToMap() {
		if job == nil || job.OverlappingUniqueIdentifier() != id {
			continue
		}
		k, _ := key.(string)
		if s._tasks.TryRemove(k) {
			s.publishSnapshot()
			return true
		}
		return false
	}
	return false
}

func (e *ScheduleEvent) describeSchedule() string {
	if e._isSchedulePerSecond {
		return fmt.Sprintf("every %ds", e._secondInterval)
	}
	return e._cronExpression
}

func (e *ScheduleEvent) jobType() string {
	if e._invocableType != nil {
		return "invocable"
	}
	return "closure"
}

func (e *ScheduleEvent) IsPaused() bool   { return e._paused }
func (e *ScheduleEvent) setPaused(p bool) { e._paused = p }
