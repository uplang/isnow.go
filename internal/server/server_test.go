package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fixedNow() time.Time {
	return time.Date(2026, 7, 14, 5, 0, 0, 0, time.UTC)
}

func newServer() http.Handler {
	return New(fixedNow, func(context.Context, time.Duration) error { return nil }).Handler()
}

func get(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestStatusEndpoint(t *testing.T) {
	h := newServer()
	if rec := get(t, h, "/v1/is/6?at=2026-01-01T06:00:00Z"); rec.Code != http.StatusNoContent {
		t.Fatalf("is (holds) = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/is/6?at=2026-01-01T07:00:00Z"); rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("is (misses) = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/is/25"); rec.Code != http.StatusBadRequest {
		t.Fatalf("is (bad isnow) = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/is/6?at=nonsense"); rec.Code != http.StatusBadRequest {
		t.Fatalf("is (bad at) = %d", rec.Code)
	}
}

func TestStatusUsesNowByDefault(t *testing.T) {
	// fixedNow is 05:00 UTC, so `5` holds with no `at`.
	if rec := get(t, newServer(), "/v1/is/5"); rec.Code != http.StatusNoContent {
		t.Fatalf("is (now) = %d", rec.Code)
	}
}

func TestCheckEndpoint(t *testing.T) {
	rec := get(t, newServer(), "/v1/check/M,W,F%20noon?at=2026-07-15T12:00:00Z")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"holds":true`) {
		t.Fatalf("check = %d %s", rec.Code, rec.Body.String())
	}
	if rec := get(t, newServer(), "/v1/check/25"); rec.Code != http.StatusBadRequest {
		t.Fatalf("check(bad) = %d", rec.Code)
	}
	if rec := get(t, newServer(), "/v1/check/6?at=bad"); rec.Code != http.StatusBadRequest {
		t.Fatalf("check(bad at) = %d", rec.Code)
	}
}

func TestNextPrevEndpoints(t *testing.T) {
	h := newServer()
	rec := get(t, h, "/v1/next/6?from=2026-07-14T07:00:00Z&n=2")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "2026-07-15T06:00:00Z") {
		t.Fatalf("next = %d %s", rec.Code, rec.Body.String())
	}
	if rec := get(t, h, "/v1/prev/6?from=2026-07-14T07:00:00Z"); rec.Code != http.StatusOK {
		t.Fatalf("prev = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/next/25"); rec.Code != http.StatusBadRequest {
		t.Fatalf("next(bad) = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/next/6?n=0"); rec.Code != http.StatusBadRequest {
		t.Fatalf("next(bad n) = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/next/6?from=bad"); rec.Code != http.StatusBadRequest {
		t.Fatalf("next(bad from) = %d", rec.Code)
	}
}

func TestCanonExplainBuild(t *testing.T) {
	h := newServer()
	if rec := get(t, h, "/v1/canon/6"); rec.Code != http.StatusOK {
		t.Fatalf("canon = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/canon/25"); rec.Code != http.StatusBadRequest {
		t.Fatalf("canon(bad) = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/explain/noon"); rec.Code != http.StatusOK {
		t.Fatalf("explain = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/explain/25"); rec.Code != http.StatusBadRequest {
		t.Fatalf("explain(bad) = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/build?weekday=M,W,F&hour=12"); rec.Code != http.StatusOK {
		t.Fatalf("build = %d", rec.Code)
	}
	if rec := get(t, h, "/v1/build?hour=25"); rec.Code != http.StatusBadRequest {
		t.Fatalf("build(bad) = %d", rec.Code)
	}
}

func TestQueryParameterAlternative(t *testing.T) {
	if rec := get(t, newServer(), "/v1/canon/?q=6"); rec.Code != http.StatusOK {
		t.Fatalf("q param = %d", rec.Code)
	}
}

func TestIndexAndNotFound(t *testing.T) {
	h := newServer()
	rec := get(t, h, "/")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "and understand date/time") {
		t.Fatalf("index = %d (len %d)", rec.Code, rec.Body.Len())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("index content-type = %q", ct)
	}
	nf := get(t, h, "/no/such/route")
	if nf.Code != http.StatusNotFound || !strings.Contains(nf.Body.String(), `"not_found"`) {
		t.Fatalf("404 = %d %q", nf.Code, nf.Body.String())
	}
}

func TestHealth(t *testing.T) {
	rec := get(t, newServer(), "/healthz")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("health = %d %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store")
	}
}
