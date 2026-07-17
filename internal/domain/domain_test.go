package domain

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	isnow "github.com/uplang/isnow.go"
	"github.com/uplang/isnow.go/internal/app"
	"github.com/uplang/isnow.go/internal/constants"
)

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("bad time %q: %v", s, err)
	}
	return ts
}

func TestQuery(t *testing.T) {
	v, err := Query("M,W,F noon", mustTime(t, "2026-07-15T12:00:00Z"))
	if err != nil || !v.Holds || v.Canonical != "*/*/* Monday,Wednesday,Friday 12:00:00" {
		t.Fatalf("Query = %+v, %v", v, err)
	}
	if _, err := Query("25", time.Now()); !errors.Is(err, isnow.ErrRange) {
		t.Fatalf("Query(bad) = %v", err)
	}
}

func TestDerive(t *testing.T) {
	occ, err := Derive(context.Background(), "6", mustTime(t, "2026-07-14T07:00:00Z"), 2, true)
	if err != nil || len(occ) != 2 {
		t.Fatalf("Derive = %v, %v", occ, err)
	}
	if _, err := Derive(context.Background(), "25", time.Now(), 1, true); !errors.Is(err, isnow.ErrRange) {
		t.Fatalf("Derive(bad) = %v", err)
	}
}

func TestDeriveStopsWhenExhausted(t *testing.T) {
	occ, err := Derive(context.Background(), "12 <2016", mustTime(t, "2020-01-01T00:00:00Z"), 3, true)
	if err != nil || len(occ) != 0 {
		t.Fatalf("Derive exhausted = %v, %v", occ, err)
	}
}

func TestCanonAndDescribe(t *testing.T) {
	c, err := Canon("6")
	if err != nil || c != "*/*/* * 06:00:00" {
		t.Fatalf("Canon = %q, %v", c, err)
	}
	if _, cerr := Canon("25"); !errors.Is(cerr, isnow.ErrRange) {
		t.Fatalf("Canon(bad) = %v", cerr)
	}
	v, err := Describe("noon")
	if err != nil || v.Explain == "" {
		t.Fatalf("Describe = %+v, %v", v, err)
	}
	if _, err := Describe("25"); !errors.Is(err, isnow.ErrRange) {
		t.Fatalf("Describe(bad) = %v", err)
	}
}

func TestBuild(t *testing.T) {
	v, src, err := Build(BuildFields{Weekday: "M,W,F", Hour: "12"})
	if err != nil || v.Canonical != "*/*/* Monday,Wednesday,Friday 12:00:00" {
		t.Fatalf("Build = %+v, src=%q, %v", v, src, err)
	}
	if _, _, err := Build(BuildFields{Hour: "25"}); !errors.Is(err, isnow.ErrRange) {
		t.Fatalf("Build(bad) = %v", err)
	}
	if _, _, err := Build(BuildFields{Hour: "12", Since: "2011", Until: "2016"}); err != nil {
		t.Fatalf("Build(bounds) = %v", err)
	}
}

// clock is a controllable time source for the effect tests.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

// stepSleep advances the clock and cancels after n calls.
func stepSleep(c *clock, n int) app.Sleeper {
	calls := 0
	return func(context.Context, time.Duration) error {
		calls++
		if calls > n {
			return context.Canceled
		}
		c.t = c.t.Add(time.Second)
		return nil
	}
}

func TestWaitSucceeds(t *testing.T) {
	c := &clock{t: mustTime(t, "2026-07-14T05:00:00Z")}
	env := &app.Env{Now: c.now, Sleep: func(context.Context, time.Duration) error { return nil }}
	if err := Wait(context.Background(), env, "6", 0); err != nil {
		t.Fatalf("Wait = %v", err)
	}
}

func TestWaitTimesOut(t *testing.T) {
	c := &clock{t: mustTime(t, "2026-07-14T07:00:00Z")}
	env := &app.Env{Now: c.now, Sleep: func(context.Context, time.Duration) error { return nil }}
	if err := Wait(context.Background(), env, "6", time.Minute); !errors.Is(err, constants.ErrTimeout) {
		t.Fatalf("Wait timeout = %v", err)
	}
}

func TestWaitBadPattern(t *testing.T) {
	env := &app.Env{Now: time.Now}
	if err := Wait(context.Background(), env, "25", 0); !errors.Is(err, isnow.ErrRange) {
		t.Fatalf("Wait(bad) = %v", err)
	}
}

func TestWaitNoOccurrence(t *testing.T) {
	c := &clock{t: mustTime(t, "2020-01-01T00:00:00Z")}
	env := &app.Env{Now: c.now}
	if err := Wait(context.Background(), env, "12 <2016", 0); !errors.Is(err, constants.ErrNoOccurrence) {
		t.Fatalf("Wait(none) = %v", err)
	}
}

func TestWaitTimeoutSleepCancelled(t *testing.T) {
	c := &clock{t: mustTime(t, "2026-07-14T07:00:00Z")}
	env := &app.Env{Now: c.now, Sleep: func(context.Context, time.Duration) error { return context.Canceled }}
	if err := Wait(context.Background(), env, "6", time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait cancelled = %v", err)
	}
}

func TestRunSpawnsAndStops(t *testing.T) {
	c := &clock{t: mustTime(t, "2026-07-14T05:00:00Z")}
	spawned := 0
	env := &app.Env{
		Now:   c.now,
		Sleep: stepSleep(c, 2),
		Spawn: func(context.Context, string, []string) error { spawned++; return nil },
	}
	entry, err := CompileEntry("*", "echo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), env, []Entry{entry}); err != nil {
		t.Fatalf("Run = %v", err)
	}
	if spawned != 2 {
		t.Fatalf("spawned %d, want 2", spawned)
	}
}

func TestRunEmpty(t *testing.T) {
	env := &app.Env{Now: time.Now}
	if err := Run(context.Background(), env, nil); !errors.Is(err, constants.ErrMissingCommand) {
		t.Fatalf("Run(empty) = %v", err)
	}
}

func TestRunNoOccurrence(t *testing.T) {
	c := &clock{t: mustTime(t, "2020-01-01T00:00:00Z")}
	env := &app.Env{Now: c.now, Sleep: func(context.Context, time.Duration) error { return nil }}
	entry, _ := CompileEntry("12 <2016", "echo", nil)
	if err := Run(context.Background(), env, []Entry{entry}); !errors.Is(err, constants.ErrNoOccurrence) {
		t.Fatalf("Run(none) = %v", err)
	}
}

func TestRunSpawnError(t *testing.T) {
	c := &clock{t: mustTime(t, "2026-07-14T05:00:00Z")}
	boom := errors.New("boom")
	env := &app.Env{
		Now:   c.now,
		Sleep: func(context.Context, time.Duration) error { return nil },
		Spawn: func(context.Context, string, []string) error { return boom },
	}
	entry, _ := CompileEntry("*", "echo", nil)
	if err := Run(context.Background(), env, []Entry{entry}); !errors.Is(err, boom) {
		t.Fatalf("Run(spawn err) = %v", err)
	}
}

func TestCompileEntryBad(t *testing.T) {
	if _, err := CompileEntry("25", "echo", nil); !errors.Is(err, isnow.ErrRange) {
		t.Fatalf("CompileEntry(bad) = %v", err)
	}
}

func TestParseNowtab(t *testing.T) {
	entries, err := ParseNowtab("# comment\n\n6\tbackup now\nM noon\tsync\n")
	if err != nil || len(entries) != 2 {
		t.Fatalf("ParseNowtab = %v, %v", entries, err)
	}
	if entries[0].Command != "backup" || len(entries[0].Args) != 1 {
		t.Fatalf("entry0 = %+v", entries[0])
	}
}

func TestParseNowtabErrors(t *testing.T) {
	if _, err := ParseNowtab("6\n"); !errors.Is(err, constants.ErrMissingCommand) {
		t.Fatalf("nowtab missing command = %v", err)
	}
	if _, err := ParseNowtab("25\tbackup\n"); !errors.Is(err, isnow.ErrRange) {
		t.Fatalf("nowtab bad isnow = %v", err)
	}
}

func TestRunGracefulOnCancel(t *testing.T) {
	c := &clock{t: mustTime(t, "2026-07-14T05:00:00Z")}
	env := &app.Env{
		Now:   c.now,
		Sleep: func(context.Context, time.Duration) error { return context.Canceled },
		Spawn: func(context.Context, string, []string) error { return nil },
	}
	entry, _ := CompileEntry("*", "echo", nil)
	if err := Run(context.Background(), env, []Entry{entry}); err != nil {
		t.Fatalf("Run graceful = %v", err)
	}
}

func TestBuildWeekdayOnly(t *testing.T) {
	v, _, err := Build(BuildFields{Weekday: "M"})
	if err != nil || v.Canonical != "*/*/* Monday *:*:00" {
		t.Fatalf("Build(weekday only) = %+v, %v", v, err)
	}
}

func TestBuildNumericWeekday(t *testing.T) {
	// A numeric weekday (Sunday = 1) must land in the weekday slot, not the hour.
	cases := []struct {
		fields BuildFields
		canon  string
	}{
		{BuildFields{Weekday: "3"}, "*/*/* Tuesday *:*:00"},
		{BuildFields{Weekday: "3", Hour: "9"}, "*/*/* Tuesday 09:00:00"},
		{BuildFields{Weekday: "2-6", Hour: "9"}, "*/*/* Monday-Friday 09:00:00"},
		{BuildFields{Weekday: "1,7"}, "*/*/* Sunday,Saturday *:*:00"},
		{BuildFields{Weekday: "M,W,F"}, "*/*/* Monday,Wednesday,Friday *:*:00"},
	}
	for _, c := range cases {
		v, _, err := Build(c.fields)
		if err != nil || v.Canonical != c.canon {
			t.Fatalf("Build(%+v) = %q, %v; want %q", c.fields, v.Canonical, err, c.canon)
		}
	}
}

func TestDerivePrev(t *testing.T) {
	occ, err := Derive(context.Background(), "6", mustTime(t, "2026-07-14T07:00:00Z"), 1, false)
	if err != nil || len(occ) != 1 || !occ[0].Equal(mustTime(t, "2026-07-14T06:00:00Z")) {
		t.Fatalf("Derive prev = %v, %v", occ, err)
	}
}

func TestRunSleepError(t *testing.T) {
	c := &clock{t: mustTime(t, "2026-07-14T05:00:00Z")}
	boom := errors.New("sleep boom")
	env := &app.Env{
		Now:   c.now,
		Sleep: func(context.Context, time.Duration) error { return boom },
		Spawn: func(context.Context, string, []string) error { return nil },
	}
	entry, _ := CompileEntry("*", "echo", nil)
	if err := Run(context.Background(), env, []Entry{entry}); !errors.Is(err, boom) {
		t.Fatalf("Run(sleep err) = %v", err)
	}
}

var _ = bytes.MinRead
