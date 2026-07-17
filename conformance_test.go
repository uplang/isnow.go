package isnow

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// corpusDir is the sibling grammar repo's conformance corpus; the suite
// self-skips when the checkout is absent (the up.js model).
const corpusDir = "../isnow/conformance"

type corpusCase struct {
	Name      string    `yaml:"name"`
	Isnow     string    `yaml:"isnow"`
	At        string    `yaml:"at"`
	From      string    `yaml:"from"`
	TZ        string    `yaml:"tz"`
	Holds     *bool     `yaml:"holds"`
	Canonical *string   `yaml:"canonical"`
	Next      *[]string `yaml:"next"`
	Prev      *[]string `yaml:"prev"`
	Error     string    `yaml:"error"`
}

func loadCorpus(t *testing.T) []corpusCase {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(corpusDir, "*.yaml"))
	if err != nil || len(files) == 0 {
		t.Skipf("conformance corpus not present at %s", corpusDir)
	}
	all := make([]corpusCase, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		var doc struct {
			Cases []corpusCase `yaml:"cases"`
		}
		if err := yaml.Unmarshal(data, &doc); err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		all = append(all, doc.Cases...)
	}
	return all
}

func TestConformance(t *testing.T) {
	for _, c := range loadCorpus(t) {
		t.Run(c.Name, func(t *testing.T) { runCase(t, c) })
	}
}

func runCase(t *testing.T, c corpusCase) {
	switch {
	case c.Error != "":
		checkError(t, c)
	case c.Canonical != nil:
		checkCanonical(t, c)
	case c.Holds != nil:
		checkHolds(t, c)
	case c.Next != nil:
		checkDerive(t, c, true)
	case c.Prev != nil:
		checkDerive(t, c, false)
	default:
		t.Fatalf("case %s has no assertion", c.Name)
	}
}

func checkError(t *testing.T, c corpusCase) {
	_, err := Parse(c.Isnow)
	if got := Code(err); got != c.Error {
		t.Fatalf("Parse(%q) error = %q, want %q", c.Isnow, got, c.Error)
	}
}

func checkCanonical(t *testing.T, c corpusCase) {
	p, err := Parse(c.Isnow)
	if err != nil {
		t.Fatalf("Parse(%q): %v", c.Isnow, err)
	}
	if p.Canonical() != *c.Canonical {
		t.Fatalf("Canonical(%q) = %q, want %q", c.Isnow, p.Canonical(), *c.Canonical)
	}
}

func checkHolds(t *testing.T, c corpusCase) {
	p, err := Parse(c.Isnow)
	if err != nil {
		t.Fatalf("Parse(%q): %v", c.Isnow, err)
	}
	if got := p.Holds(mustTime(t, c.At)); got != *c.Holds {
		t.Fatalf("Holds(%q, %s) = %v, want %v", c.Isnow, c.At, got, *c.Holds)
	}
}

func checkDerive(t *testing.T, c corpusCase, forward bool) {
	p, err := Parse(c.Isnow)
	if err != nil {
		t.Fatalf("Parse(%q): %v", c.Isnow, err)
	}
	want := c.Next
	if !forward {
		want = c.Prev
	}
	got := deriveN(p, mustTime(t, c.From), len(*want), forward)
	assertInstants(t, c, got, *want)
}

func assertInstants(t *testing.T, c corpusCase, got []time.Time, want []string) {
	if len(got) != len(want) {
		t.Fatalf("derive(%q) got %d occurrences, want %d", c.Isnow, len(got), len(want))
	}
	for i := range want {
		if !got[i].Equal(mustTime(t, want[i])) {
			t.Fatalf("derive(%q)[%d] = %s, want %s", c.Isnow, i, got[i].Format(time.RFC3339), want[i])
		}
	}
}

func deriveN(p Pattern, from time.Time, n int, forward bool) []time.Time {
	out := make([]time.Time, 0, n)
	cur := from
	for len(out) < n {
		next, ok := step(p, cur, forward)
		if !ok {
			break
		}
		out = append(out, next)
		cur = next
	}
	return out
}

func step(p Pattern, from time.Time, forward bool) (time.Time, bool) {
	if forward {
		return p.Next(from)
	}
	return p.Prev(from)
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad time %q: %v", s, err)
	}
	return ts
}
