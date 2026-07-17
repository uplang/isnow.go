package server

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"time"
)

const (
	defaultWaitTimeout = 60 * time.Second
	maxWaitTimeout     = 10 * time.Minute
)

func (s *Server) handleWait(w http.ResponseWriter, r *http.Request) {
	p, ok := parsePattern(w, isnowOf(r))
	if !ok {
		return
	}
	timeout, ok := timeoutOr400(w, r)
	if !ok {
		return
	}
	now := s.now()
	next, has, err := s.nextWithin(r, p, now)
	if err != nil {
		writeDeriveError(w, err)
		return
	}
	if !has || next.Sub(now) > timeout {
		s.waitThenExpire(w, r, has, timeout)
		return
	}
	if err := s.sleep(r.Context(), next.Sub(now)); err != nil {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// nextWithin derives the next occurrence under the derivation time budget, so a
// wait/watch on a pathological pattern cannot pin a core.
func (s *Server) nextWithin(r *http.Request, p patternNext, now time.Time) (time.Time, bool, error) {
	ctx, cancel := context.WithTimeout(r.Context(), deriveBudget)
	defer cancel()
	return p.NextContext(ctx, now)
}

// waitThenExpire sleeps out the timeout (when there is no in-window occurrence)
// and responds 504.
func (s *Server) waitThenExpire(w http.ResponseWriter, r *http.Request, has bool, timeout time.Duration) {
	if has {
		if err := s.sleep(r.Context(), timeout); err != nil {
			return
		}
	}
	w.WriteHeader(http.StatusGatewayTimeout)
}

func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	p, ok := parsePattern(w, isnowOf(r))
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	s.streamOccurrences(r, w, flusher, p)
}

func (s *Server) streamOccurrences(r *http.Request, w http.ResponseWriter, flusher http.Flusher, p patternNext) {
	for {
		now := s.now()
		next, has, err := s.nextWithin(r, p, now)
		if err != nil || !has {
			return
		}
		if err := s.sleep(r.Context(), next.Sub(now)); err != nil {
			return
		}
		_, _ = fmt.Fprintf(w, "event: occurrence\ndata: %s\n\n", html.EscapeString(next.Format(time.RFC3339)))
		flusher.Flush()
	}
}

// patternNext is the derivation slice of a Pattern the stream needs.
type patternNext interface {
	NextContext(ctx context.Context, from time.Time) (time.Time, bool, error)
}

func timeoutOr400(w http.ResponseWriter, r *http.Request) (time.Duration, bool) {
	raw := r.URL.Query().Get("timeout")
	if raw == "" {
		return defaultWaitTimeout, true
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 || d > maxWaitTimeout {
		writeError(w, http.StatusBadRequest, "parameter", "timeout must be 0..10m")
		return 0, false
	}
	return d, true
}
