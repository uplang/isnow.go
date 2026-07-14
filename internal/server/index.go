package server

import (
	_ "embed"
	"net/http"
)

// indexPage is the interactive single-page app served at the root: a live
// isnow evaluator and builder that calls this server's own /v1 API.
//
//go:embed index.html
var indexPage string

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(indexPage))
}

// handleNotFound is the catch-all for unmatched routes: the contract's JSON error.
func (s *Server) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
}
