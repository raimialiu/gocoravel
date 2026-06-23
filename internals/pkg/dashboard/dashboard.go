// Package dashboard serves a Bull/Temporal-style web UI for a coravel scheduler:
// it visualizes jobs and run history and exposes controls (trigger, pause/resume,
// unschedule, replay). The handler is mountable on any net/http server, and the
// UI updates live over Server-Sent Events.
package dashboard

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/raimialiu/gocoravel/internals/pkg/scheduler"
)

//go:embed index.html
var indexHTML string

// Controller is the slice of the scheduler the dashboard drives. *scheduler.Scheduler
// satisfies it.
type Controller interface {
	Jobs() []scheduler.JobView
	History(id string) []scheduler.JobRun
	Metrics(id string) scheduler.JobMetrics
	Overview() scheduler.Overview
	Errors(limit int) []scheduler.JobRun
	TriggerNow(id string) error
	Pause(id string) error
	Resume(id string) error
	Unschedule(id string) bool
	ClearHistory(id string)
	Replay() error
	Snapshot() scheduler.DashboardEvent
	Subscribe() (<-chan scheduler.DashboardEvent, func())
}

// Options configures the dashboard handler.
type Options struct {
	Addr     string // listen address for coravel.AddDashboard (default ":8099")
	BasePath string // path the UI + API are served under (default "/coravel")
	Title    string // shown in the UI (default "Coravel")
	Token    string // if set, requests must carry ?token= or "Authorization: Bearer <token>"
}

// Handler returns a mountable http.Handler. Mount it on your own mux at BasePath
// (e.g. mux.Handle("/coravel/", h)) — the routes already include BasePath, so do
// not strip the prefix.
func Handler(ctrl Controller, opts Options) http.Handler {
	base := strings.TrimRight(opts.BasePath, "/")
	if base == "" {
		base = "/coravel"
	}
	if opts.Title == "" {
		opts.Title = "Coravel"
	}
	page := strings.NewReplacer("{{BASE}}", base, "{{TITLE}}", opts.Title).Replace(indexHTML)

	mux := http.NewServeMux()

	mux.HandleFunc("GET "+base+"/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, page)
	})

	mux.HandleFunc("GET "+base+"/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, ctrl.Jobs())
	})
	mux.HandleFunc("GET "+base+"/api/jobs/{id}/runs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, ctrl.History(r.PathValue("id")))
	})
	mux.HandleFunc("GET "+base+"/api/jobs/{id}/metrics", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, ctrl.Metrics(r.PathValue("id")))
	})
	mux.HandleFunc("DELETE "+base+"/api/jobs/{id}/runs", func(w http.ResponseWriter, r *http.Request) {
		ctrl.ClearHistory(r.PathValue("id"))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET "+base+"/api/overview", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, ctrl.Overview())
	})
	mux.HandleFunc("GET "+base+"/api/errors", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, ctrl.Errors(50))
	})
	mux.HandleFunc("POST "+base+"/api/jobs/{id}/trigger", actionHandler(ctrl.TriggerNow))
	mux.HandleFunc("POST "+base+"/api/jobs/{id}/pause", actionHandler(ctrl.Pause))
	mux.HandleFunc("POST "+base+"/api/jobs/{id}/resume", actionHandler(ctrl.Resume))
	mux.HandleFunc("DELETE "+base+"/api/jobs/{id}", func(w http.ResponseWriter, r *http.Request) {
		if ctrl.Unschedule(r.PathValue("id")) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "job not found", http.StatusNotFound)
	})
	mux.HandleFunc("POST "+base+"/api/replay", func(w http.ResponseWriter, r *http.Request) {
		if err := ctrl.Replay(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET "+base+"/api/stream", streamHandler(ctrl))

	return tokenAuth(opts.Token, mux)
}

func actionHandler(fn func(string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(r.PathValue("id")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func streamHandler(ctrl Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		h := w.Header()
		h.Set("Content-Type", "text/event-stream")
		h.Set("Cache-Control", "no-cache")
		h.Set("Connection", "keep-alive")

		events, cancel := ctrl.Subscribe()
		defer cancel()

		writeEvent(w, ctrl.Snapshot())
		flusher.Flush()

		ping := time.NewTicker(15 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case ev, ok := <-events:
				if !ok {
					return
				}
				writeEvent(w, ev)
				flusher.Flush()
			case <-ping.C:
				_, _ = io.WriteString(w, ": ping\n\n")
				flusher.Flush()
			}
		}
	}
}

func writeEvent(w io.Writer, ev scheduler.DashboardEvent) {
	b, err := json.Marshal(ev)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", b)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func tokenAuth(token string, next http.Handler) http.Handler {
	if token == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := r.URL.Query().Get("token")
		if provided == "" {
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				provided = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		if provided != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
