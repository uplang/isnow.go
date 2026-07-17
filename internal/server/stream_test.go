package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// steppingClock advances by each sleep duration and cancels after n sleeps.
type steppingClock struct {
	t     time.Time
	calls int
	limit int
}

func (s *steppingClock) now() time.Time { return s.t }

func (s *steppingClock) sleep(context.Context, time.Duration) error {
	s.calls++
	if s.calls > s.limit {
		return context.Canceled
	}
	s.t = s.t.Add(time.Hour)
	return nil
}

func TestWaitSucceeds(t *testing.T) {
	// `*` next occurs at 05:01 (one minute from 05:00), within the timeout.
	srv := New(fixedNow, func(context.Context, time.Duration) error { return nil })
	rec := get(t, srv.Handler(), "/v1/wait/*?timeout=5m")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("wait = %d", rec.Code)
	}
}

func TestWaitSleepCancelled(t *testing.T) {
	srv := New(fixedNow, func(context.Context, time.Duration) error { return context.Canceled })
	rec := get(t, srv.Handler(), "/v1/wait/*?timeout=5m")
	if rec.Code != http.StatusOK { // no body written on cancel
		t.Fatalf("wait cancelled = %d", rec.Code)
	}
}

func TestWaitExpireSleepCancelled(t *testing.T) {
	srv := New(fixedNow, func(context.Context, time.Duration) error { return context.Canceled })
	rec := get(t, srv.Handler(), "/v1/wait/6?timeout=5m")
	if rec.Code != http.StatusOK { // cancelled before the timeout elapses
		t.Fatalf("wait expire cancelled = %d", rec.Code)
	}
}

func TestWaitTimesOut(t *testing.T) {
	srv := New(fixedNow, func(context.Context, time.Duration) error { return nil })
	rec := get(t, srv.Handler(), "/v1/wait/6?timeout=1m")
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("wait timeout = %d", rec.Code)
	}
}

func TestWaitNoOccurrence(t *testing.T) {
	srv := New(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }, nil)
	rec := get(t, srv.Handler(), "/v1/wait/12%20%3C2016")
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("wait none = %d", rec.Code)
	}
}

func TestWaitBadInputs(t *testing.T) {
	srv := New(fixedNow, nil)
	if rec := get(t, srv.Handler(), "/v1/wait/25"); rec.Code != http.StatusBadRequest {
		t.Fatalf("wait bad isnow = %d", rec.Code)
	}
	if rec := get(t, srv.Handler(), "/v1/wait/6?timeout=99h"); rec.Code != http.StatusBadRequest {
		t.Fatalf("wait bad timeout = %d", rec.Code)
	}
}

func TestWatchStreams(t *testing.T) {
	clk := &steppingClock{t: fixedNow(), limit: 2}
	srv := New(clk.now, clk.sleep)
	rec := get(t, srv.Handler(), "/v1/watch/6")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "event: occurrence") {
		t.Fatalf("watch = %d %s", rec.Code, rec.Body.String())
	}
}

func TestWatchBadPattern(t *testing.T) {
	srv := New(fixedNow, nil)
	if rec := get(t, srv.Handler(), "/v1/watch/25"); rec.Code != http.StatusBadRequest {
		t.Fatalf("watch bad = %d", rec.Code)
	}
}

func TestWatchNoOccurrence(t *testing.T) {
	srv := New(func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
		func(context.Context, time.Duration) error { return nil })
	rec := get(t, srv.Handler(), "/v1/watch/12%20%3C2016")
	if rec.Code != http.StatusOK {
		t.Fatalf("watch none = %d", rec.Code)
	}
}

// nonFlusher is a ResponseWriter without Flush, to exercise the SSE guard.
type nonFlusher struct{ h http.Header }

func (n nonFlusher) Header() http.Header       { return n.h }
func (nonFlusher) Write(b []byte) (int, error) { return len(b), nil }
func (nonFlusher) WriteHeader(int)             {}

func TestWatchStreamingUnsupported(_ *testing.T) {
	srv := New(fixedNow, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/watch/6", nil)
	req.SetPathValue("isnow", "6")
	srv.handleWatch(nonFlusher{h: http.Header{}}, req)
}
