// Package server implements the isnow HTTP API (specs/contracts/http-api.md):
// a status-code membership test, occurrence derivation, a builder, long-poll,
// and SSE — every handler a pure function of (request, injected clock).
package server

import (
	"net/http"
	"time"

	"github.com/uplang/isnow.go/internal/app"
)

// Server serves the isnow HTTP API with an injected clock and sleeper.
type Server struct {
	now   app.Clock
	sleep app.Sleeper
}

// New builds a Server. sleep drives wait/watch timing (injectable for tests).
func New(now app.Clock, sleep app.Sleeper) *Server {
	return &Server{now: now, sleep: sleep}
}

// Handler returns the routed HTTP handler with no-store caching.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/is/{isnow...}", s.handleIs)
	mux.HandleFunc("GET /v1/check/{isnow...}", s.handleCheck)
	mux.HandleFunc("GET /v1/next/{isnow...}", s.handleNext)
	mux.HandleFunc("GET /v1/prev/{isnow...}", s.handlePrev)
	mux.HandleFunc("GET /v1/canon/{isnow...}", s.handleCanon)
	mux.HandleFunc("GET /v1/explain/{isnow...}", s.handleExplain)
	mux.HandleFunc("GET /v1/build", s.handleBuild)
	mux.HandleFunc("GET /v1/wait/{isnow...}", s.handleWait)
	mux.HandleFunc("GET /v1/watch/{isnow...}", s.handleWatch)
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("/", s.handleNotFound)
	return noStore(mux)
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// isnowOf reads the isnow from the path wildcard or the q query parameter.
func isnowOf(r *http.Request) string {
	if v := r.PathValue("isnow"); v != "" {
		return v
	}
	return r.URL.Query().Get("q")
}

// instant reads the `at`/`from` parameter (RFC 3339) or defaults to now.
func (s *Server) instant(r *http.Request, key string) (time.Time, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return s.now(), nil
	}
	return time.Parse(time.RFC3339, v)
}
