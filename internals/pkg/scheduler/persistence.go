package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/adhocore/gronx"
	"github.com/raimialiu/gocoravel/internals/pkg/store"
	"github.com/raimialiu/gocoravel/internals/pkg/store/providers"
)

// RunStatus is the terminal state of a single job execution.
type RunStatus string

const (
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	RunSkipped   RunStatus = "skipped"
)

// JobSchedule is the persistable description of a scheduled job. It captures the
// schedule and identity — never the Go func/closure, which cannot be serialized.
// Closures are restored by the app re-declaring them under the same UniqueId;
// named invocables are rebuilt from the invocable registry via InvocableType.
type JobSchedule struct {
	UniqueId       string `json:"uniqueId"`
	CronExpression string `json:"cronExpression"`
	IsSecondBased  bool   `json:"isSecondBased"`
	SecondInterval int    `json:"secondInterval"`
	RunOnce        bool   `json:"runOnce"`
	Location       string `json:"location"`
	InvocableType  string `json:"invocableType"` // non-empty => rebuildable from the registry
	Enabled        bool   `json:"enabled"`
}

// JobRun is one execution record — the unit of run history (and log).
type JobRun struct {
	Id           string    `json:"id"`
	JobId        string    `json:"jobId"`
	ScheduledFor time.Time `json:"scheduledFor"`
	StartedAt    time.Time `json:"startedAt"`
	FinishedAt   time.Time `json:"finishedAt"`
	Status       RunStatus `json:"status"`
	Err          string    `json:"err,omitempty"`
	Logs         []string  `json:"logs,omitempty"`
}

const (
	scheduleKeyPrefix = "coravel:schedule:"
	scheduleIndexKey  = "coravel:schedules"
	runsKeyPrefix     = "coravel:runs:"

	// autoIDPrefix marks a provisional id assigned at creation (so unnamed jobs
	// never collide on the empty key). Persistence replaces it with a stable id.
	autoIDPrefix = "__auto:"

	// backfill bounds — keep replay from becoming a thundering herd or scanning
	// pathologically long downtimes.
	maxBackfillRuns  = 500
	maxBackfillScans = 100_000
)

func scheduleKey(id string) string { return scheduleKeyPrefix + id }
func runsKey(id string) string     { return runsKeyPrefix + id }

// -----------------------------------------------------------------------------
// Store wiring + error reporting
// -----------------------------------------------------------------------------

// UseStore attaches a persistence store. With a store set, the scheduler can
// persist job schedules and run history, and replay them on restart.
func (s *Scheduler) UseStore(st *store.CoravelStore) { s._store = st }

// PersistenceEnabled reports whether a persistence store is attached.
func (s *Scheduler) PersistenceEnabled() bool { return s._store != nil }

// OnError registers a handler invoked for non-fatal errors (e.g. a persistence
// write that fails). Without one, such errors are swallowed.
func (s *Scheduler) OnError(handler func(error)) { s._errorHandler = handler }

func (s *Scheduler) reportError(err error) {
	if err == nil || s._errorHandler == nil {
		return
	}
	s._errorHandler(err)
}

// -----------------------------------------------------------------------------
// Invocable registry (for replay of named invocables)
// -----------------------------------------------------------------------------

var invocableRegistry = map[string]func() IInvocable{}

// RegisterInvocable registers a constructor for an IInvocable type so it can be
// rebuilt and rescheduled on restart without the app re-declaring it. The key is
// derived from sample's concrete type, so register with a value of the same type
// you schedule.
func RegisterInvocable(sample IInvocable, constructor func() IInvocable) {
	if name := invocableName(sample); name != "" {
		invocableRegistry[name] = constructor
	}
}

func invocableName(inv IInvocable) string {
	t := reflect.TypeOf(inv)
	if t == nil {
		return ""
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() + "." + t.Name()
}

// -----------------------------------------------------------------------------
// Identity — stable ids for persistent jobs
// -----------------------------------------------------------------------------

func (e *ScheduleEvent) hasAutoID() bool { return strings.HasPrefix(e._uniqueId, autoIDPrefix) }

// deriveStableID produces an id that is the same across restarts for the same
// job: a named invocable uses its type name; a closure uses its func location
// plus its schedule signature.
func (e *ScheduleEvent) deriveStableID() string {
	if e._invocableType != nil {
		if name := invocableName(*e._invocableType); name != "" {
			return name
		}
	}
	base := e._scheduleAction.FuncName()
	if base == "" {
		base = "job"
	}
	return base + "@" + e.scheduleSignature()
}

func (e *ScheduleEvent) scheduleSignature() string {
	if e._isSchedulePerSecond {
		return fmt.Sprintf("every-%ds", e._secondInterval)
	}
	return "cron-" + e._cronExpression
}

// -----------------------------------------------------------------------------
// Persistence — writes
// -----------------------------------------------------------------------------

// Sync persists the schedules of all currently-registered jobs that opted in via
// EnsurePersistence. It also assigns a stable id to any persistent job that lacks
// an explicit Name and re-keys the task table to match. Replay calls this for you.
func (s *Scheduler) Sync() {
	if s._store == nil {
		return
	}
	for key, job := range s._tasks.ToMap() {
		if job == nil || !job.ShouldPersist() {
			continue
		}
		if job._uniqueId == "" || job.hasAutoID() {
			job._uniqueId = job.deriveStableID()
		}
		if k, ok := key.(string); ok && k != job._uniqueId {
			s._tasks.TryRemove(k)
			s._tasks.TryAdd(job._uniqueId, job)
		}
		s.persistSchedule(job)
	}
}

func (s *Scheduler) persistSchedule(e *ScheduleEvent) {
	if s._store == nil || e == nil || !e.ShouldPersist() {
		return
	}
	sched := scheduleFromEvent(e)
	s.reportError(s._store.DataSource.Create(scheduleKey(sched.UniqueId), sched))
	s.indexSchedule(sched.UniqueId)
}

func (s *Scheduler) persistRun(run JobRun) {
	if s._store == nil {
		return
	}
	s.reportError(s._store.DataSource.CreateMany(runsKey(run.JobId), run))
}

func (s *Scheduler) indexSchedule(id string) {
	ids, err := s.loadScheduleIDs()
	if err != nil {
		s.reportError(err)
		return
	}
	for _, existing := range ids {
		if existing == id {
			return
		}
	}
	s.reportError(s._store.DataSource.CreateMany(scheduleIndexKey, id))
}

func scheduleFromEvent(e *ScheduleEvent) JobSchedule {
	sched := JobSchedule{
		UniqueId:       e._uniqueId,
		CronExpression: e._cronExpression,
		IsSecondBased:  e._isSchedulePerSecond,
		SecondInterval: e._secondInterval,
		RunOnce:        e._runOnce,
		Location:       e._preferredLocation,
		Enabled:        true,
	}
	if e._invocableType != nil {
		sched.InvocableType = invocableName(*e._invocableType)
	}
	return sched
}

// -----------------------------------------------------------------------------
// Persistence — reads
// -----------------------------------------------------------------------------

func (s *Scheduler) loadScheduleIDs() ([]string, error) {
	raw, err := s._store.DataSource.Get(scheduleIndexKey)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(raw))
	for _, v := range raw {
		if str, ok := v.(string); ok {
			ids = append(ids, str)
		}
	}
	return ids, nil
}

func (s *Scheduler) loadSchedules() ([]JobSchedule, error) {
	ids, err := s.loadScheduleIDs()
	if err != nil {
		return nil, err
	}
	out := make([]JobSchedule, 0, len(ids))
	for _, id := range ids {
		v, err := s._store.DataSource.GetSingle(scheduleKey(id))
		if errors.Is(err, providers.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var sched JobSchedule
		if err := remarshal(v, &sched); err != nil {
			return nil, err
		}
		out = append(out, sched)
	}
	return out, nil
}

func (s *Scheduler) loadRuns(jobID string) ([]JobRun, error) {
	raw, err := s._store.DataSource.Get(runsKey(jobID))
	if err != nil {
		return nil, err
	}
	out := make([]JobRun, 0, len(raw))
	for _, v := range raw {
		var run JobRun
		if err := remarshal(v, &run); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, nil
}

// remarshal round-trips a generically-decoded value (e.g. map[string]interface{})
// into a typed struct, since DataProvider returns untyped JSON shapes.
func remarshal(src, dst interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// -----------------------------------------------------------------------------
// Execution + run recording
// -----------------------------------------------------------------------------

// execute runs a due job once, recording a JobRun when the job opted into
// persistence. It runs synchronously within its goroutine so completion and
// failure can be captured.
func (s *Scheduler) execute(job *ScheduleEvent, t time.Time) {
	if job == nil || job.WhenPredicateFails() {
		return
	}

	started := time.Now()
	err := job.runNow()
	finished := time.Now()

	job.markAsExecuteOnce()

	run := JobRun{
		Id:           fmt.Sprintf("%s:%d", job._uniqueId, started.UnixNano()),
		JobId:        job.OverlappingUniqueIdentifier(),
		ScheduledFor: t,
		StartedAt:    started,
		FinishedAt:   finished,
		Status:       RunSucceeded,
	}
	if err != nil {
		run.Status = RunFailed
		run.Err = err.Error()
		s.reportError(err)
	}

	// Always record to the in-memory feed (dashboard), persist only if opted in.
	if s._feed != nil {
		s._feed.record(run)
	}
	if s._store != nil && job.ShouldPersist() {
		s.persistRun(run)
	}

	job.unscheduleIfWarranted()
}

// runNow executes the job's action synchronously, recovering from panics so a
// failing job is recorded rather than crashing the scheduler goroutine.
func (e *ScheduleEvent) runNow() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("scheduler: job %q panicked: %v", e._uniqueId, r)
		}
	}()

	if e._invocableType == nil {
		e._scheduleAction.Run()
		return nil
	}

	invocable := *e._invocableType
	if len(e._parameters) == 0 {
		_, err = invocable.Invoke()
		return err
	}
	_, err = invocable.InvokeWithPayload(e._parameters...)
	return err
}

// -----------------------------------------------------------------------------
// Replay + backfill
// -----------------------------------------------------------------------------

// Replay reconciles persisted state with the in-memory jobs after a restart:
//   - persists currently-registered schedules (Sync),
//   - rebuilds registered invocables that weren't re-declared and reschedules them,
//   - restores run-once state so already-run one-shot jobs don't run again,
//   - backfills the runs each job missed while down (bounded — see backfill).
//
// Call Replay after registering your jobs.
func (s *Scheduler) Replay() error {
	if s._store == nil {
		return errors.New("scheduler: persistence not enabled")
	}

	s.Sync()

	schedules, err := s.loadSchedules()
	if err != nil {
		return err
	}

	for _, sched := range schedules {
		job := s.lookup(sched.UniqueId)
		if job == nil && sched.InvocableType != "" {
			if constructor, ok := invocableRegistry[sched.InvocableType]; ok {
				job = s.scheduleFromState(constructor(), sched)
			}
		}
		if job == nil {
			// A closure that wasn't re-declared — cannot be rebuilt from storage.
			continue
		}

		runs, err := s.loadRuns(sched.UniqueId)
		if err != nil {
			s.reportError(err)
			continue
		}
		if sched.RunOnce && ranSuccessfully(runs) {
			job.markAsExecuteOnce()
			job.unscheduleIfWarranted()
			continue
		}
		go s.backfill(job, sched, lastSuccessful(runs))
	}
	return nil
}

// backfill replays the occurrences a job missed between its last successful run
// and now, in order. It is bounded two ways: the scan window covers at most
// maxBackfillScans steps (so a long downtime can't loop forever), and at most
// maxBackfillRuns occurrences are actually replayed (most recent kept) so a
// high-frequency job can't stampede. A never-run job is caught up once.
func (s *Scheduler) backfill(job *ScheduleEvent, sched JobSchedule, last time.Time) {
	occurrences := job.missedOccurrences(sched, last, time.Now())
	if len(occurrences) > maxBackfillRuns {
		s.reportError(fmt.Errorf(
			"scheduler: %q missed %d runs while down; replaying the most recent %d",
			sched.UniqueId, len(occurrences), maxBackfillRuns,
		))
		occurrences = occurrences[len(occurrences)-maxBackfillRuns:]
	}
	for _, occurrence := range occurrences {
		s.execute(job, occurrence)
	}
}

func (e *ScheduleEvent) missedOccurrences(sched JobSchedule, last, now time.Time) []time.Time {
	if last.IsZero() {
		return []time.Time{now} // never ran: a single catch-up baseline
	}

	loc := e._timeLocation
	if loc == nil {
		loc = time.UTC
	}

	step := time.Second
	if !sched.IsSecondBased {
		step = time.Minute
	}

	start := last.Add(step).Truncate(step)
	if earliest := now.Add(-time.Duration(maxBackfillScans) * step); start.Before(earliest) {
		start = earliest
	}

	g := gronx.New()
	occurrences := make([]time.Time, 0)
	for t := start; !t.After(now); t = t.Add(step) {
		if sched.IsSecondBased {
			if sched.SecondInterval > 0 && t.Second()%sched.SecondInterval == 0 {
				occurrences = append(occurrences, t)
			}
			continue
		}
		if sched.CronExpression == "" {
			break
		}
		if due, err := g.IsDue(sched.CronExpression, t.In(loc)); err == nil && due {
			occurrences = append(occurrences, t)
		}
	}
	return occurrences
}

func (s *Scheduler) lookup(id string) *ScheduleEvent {
	// Scan by unique id rather than map key: an unnamed/late-named job may be
	// keyed under its provisional auto id, so Get(id) would miss it.
	for _, job := range s._tasks.ToMap() {
		if job != nil && job.OverlappingUniqueIdentifier() == id {
			return job
		}
	}
	return nil
}

func (s *Scheduler) scheduleFromState(invocable IInvocable, sched JobSchedule) *ScheduleEvent {
	ev := NewScheduleEvent(WithInvocableType(invocable))
	ev._uniqueId = sched.UniqueId
	ev._cronExpression = sched.CronExpression
	ev._isSchedulePerSecond = sched.IsSecondBased
	ev._secondInterval = sched.SecondInterval
	ev._runOnce = sched.RunOnce
	ev._ensurePersistence = true
	ev._scheduler = s
	if sched.Location != "" {
		ev._preferredLocation = sched.Location
		if loc, err := time.LoadLocation(sched.Location); err == nil {
			ev._timeLocation = loc
		}
	}
	s._tasks.TryAdd(ev._uniqueId, ev)
	return ev
}

func ranSuccessfully(runs []JobRun) bool {
	for _, r := range runs {
		if r.Status == RunSucceeded {
			return true
		}
	}
	return false
}

func lastSuccessful(runs []JobRun) time.Time {
	var last time.Time
	for _, r := range runs {
		if r.Status == RunSucceeded && r.FinishedAt.After(last) {
			last = r.FinishedAt
		}
	}
	return last
}
