package server

import (
	"encoding/json"
	"net/http"

	isnow "github.com/tsvsheet/go-isnow"
)

// checkBody is the JSON of the /v1/check response.
type checkBody struct {
	Isnow     string `json:"isnow"`
	Canonical string `json:"canonical"`
	At        string `json:"at,omitempty"`
	Holds     bool   `json:"holds"`
}

// occurrencesBody is the JSON of /v1/next and /v1/prev.
type occurrencesBody struct {
	Isnow       string   `json:"isnow"`
	Canonical   string   `json:"canonical"`
	Occurrences []string `json:"occurrences"`
}

// describeBody is the JSON of /v1/canon, /v1/explain, and /v1/build.
type describeBody struct {
	Isnow       string `json:"isnow"`
	Canonical   string `json:"canonical"`
	Explanation string `json:"explanation,omitempty"`
}

// errorBody is the JSON error envelope.
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: message}})
}

// parsePattern parses src or writes a 400 with the library error code.
func parsePattern(w http.ResponseWriter, src string) (isnow.Pattern, bool) {
	p, err := isnow.Parse(isnow.PatternText(src))
	if err != nil {
		writeError(w, http.StatusBadRequest, isnow.Code(err), err.Error())
		return isnow.Pattern{}, false
	}
	return p, true
}

func isnowCode(err error) string { return isnow.Code(err) }

// canonOf returns the canonical form of an already-validated src.
func canonOf(src string) string {
	p, _ := isnow.Parse(isnow.PatternText(src))
	return p.Canonical()
}
