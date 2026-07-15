package isnow

import (
	"errors"
	"testing"
	"time"
)

// holdsCase drives a table of instants against one pattern.
func holdsCase(t *testing.T, pat string, cases map[string]bool) {
	t.Helper()
	p := mustParse(t, pat)
	for ts, want := range cases {
		if got := p.Holds(at(t, ts)); got != want {
			t.Errorf("%q Holds(%s) = %v, want %v", pat, ts, got, want)
		}
	}
}

// --- container: day (sub-day strides re-align at midnight) ---

func TestIntervalEveryNMinutes(t *testing.T) {
	if got := mustParse(t, "+[90mn]").Canonical(); got != "*/*/* * *:*:00 +[90mn]" {
		t.Fatalf("Canonical = %q", got)
	}
	// 90 divides 1440, so the day container has no ragged boundary.
	holdsCase(t, "+[90mn]", map[string]bool{
		"2026-07-14T00:00:00Z": true,
		"2026-07-14T01:30:00Z": true,
		"2026-07-14T03:00:00Z": true,
		"2026-07-14T22:30:00Z": true,
		"2026-07-14T01:00:00Z": false,
		"2026-07-14T02:00:00Z": false, // 120 mod 90 = 30
		"2026-07-14T00:00:30Z": false, // off the minute boundary
	})
}

func TestIntervalEveryNHoursWithinDay(t *testing.T) {
	holdsCase(t, "+[2h]", map[string]bool{
		"2026-07-14T00:00:00Z": true,
		"2026-07-14T02:00:00Z": true,
		"2026-07-14T22:00:00Z": true,
		"2026-07-14T01:00:00Z": false,
		"2026-07-14T23:00:00Z": false,
		"2026-07-14T02:30:00Z": false, // off the hour boundary
	})
}

func TestIntervalSecondsWithinMinute(t *testing.T) {
	holdsCase(t, "+[30s]", map[string]bool{
		"2026-07-14T09:00:00Z": true,
		"2026-07-14T09:00:30Z": true,
		"2026-07-14T09:00:15Z": false,
		"2026-07-14T09:00:45Z": false,
	})
}

// --- container: week (day/hour strides re-align every Sunday) ---

func TestIntervalEveryNDaysWithinWeek(t *testing.T) {
	// +[3d] → week container, Sunday start: weekday ∈ {Sun, Wed, Sat}.
	holdsCase(t, "+[3d]", map[string]bool{
		"2026-07-12T00:00:00Z": true,  // Sunday
		"2026-07-15T00:00:00Z": true,  // Wednesday
		"2026-07-18T00:00:00Z": true,  // Saturday
		"2026-07-19T00:00:00Z": true,  // next Sunday — re-aligned
		"2026-07-13T00:00:00Z": false, // Monday
		"2026-07-14T00:00:00Z": false, // Tuesday
		"2026-07-16T00:00:00Z": false, // Thursday
		"2026-07-12T06:00:00Z": false, // right weekday, wrong time-of-day
	})
}

func TestIntervalWeekBoundaryGap(t *testing.T) {
	// Saturday (weekday 7) holds, then the next hold is Sunday (weekday 1): a
	// one-day gap, shorter than the 3-day stride — the civil re-alignment.
	p := mustParse(t, "+[3d]")
	sat := at(t, "2026-07-18T00:00:00Z")
	next, ok := p.Next(sat)
	if !ok || !next.Equal(at(t, "2026-07-19T00:00:00Z")) {
		t.Fatalf("Next after Saturday = %s (%v), want the following Sunday", next.Format(time.RFC3339), ok)
	}
}

func TestIntervalEvery25HoursWalksTheWeek(t *testing.T) {
	// Hour-of-week ≡ 0 mod 25 draws a diagonal that advances an hour per day and
	// re-aligns each Sunday 00:00.
	holdsCase(t, "+[25h]", map[string]bool{
		"2026-07-12T00:00:00Z": true,  // Sun 00:00 (hour-of-week 0)
		"2026-07-13T01:00:00Z": true,  // Mon 01:00 (25)
		"2026-07-14T02:00:00Z": true,  // Tue 02:00 (50)
		"2026-07-15T03:00:00Z": true,  // Wed 03:00 (75)
		"2026-07-16T04:00:00Z": true,  // Thu 04:00 (100)
		"2026-07-17T05:00:00Z": true,  // Fri 05:00 (125)
		"2026-07-18T06:00:00Z": true,  // Sat 06:00 (150)
		"2026-07-14T00:00:00Z": false, // Tue 00:00 (48)
		"2026-07-16T00:00:00Z": false, // Thu 00:00 (96)
		"2026-07-14T02:30:00Z": false, // off the hour boundary
	})
}

// --- container: month (day strides 8..31 re-align on the 1st) ---

func TestIntervalEveryNDaysWithinMonth(t *testing.T) {
	// +[10d] → month container: days 1, 11, 21, 31.
	holdsCase(t, "+[10d]", map[string]bool{
		"2026-07-01T00:00:00Z": true,
		"2026-07-11T00:00:00Z": true,
		"2026-07-21T00:00:00Z": true,
		"2026-07-31T00:00:00Z": true,  // 31-day month reaches the 31st
		"2026-08-01T00:00:00Z": true,  // re-aligned on the next 1st
		"2026-07-16T00:00:00Z": false, // day 16
		"2026-07-11T12:00:00Z": false, // right day, wrong time-of-day
	})
}

func TestIntervalMonthBoundaryGapAcrossShortMonth(t *testing.T) {
	// February (28 days in 2026) never reaches day 31, and March re-aligns on
	// the 1st — the last stride of a month can be short.
	holdsCase(t, "+[10d]", map[string]bool{
		"2026-02-21T00:00:00Z": true,  // last hold in February
		"2026-02-28T00:00:00Z": false, // day 28 is not on the grid
		"2026-03-01T00:00:00Z": true,  // re-aligned (7 days after Feb 21)
	})
}

// --- container: year (day strides > 31 re-align on Jan 1) ---

func TestIntervalEveryNDaysWithinYear(t *testing.T) {
	// +[40d] → year container: day-of-year 1, 41, 81, ….
	holdsCase(t, "+[40d]", map[string]bool{
		"2026-01-01T00:00:00Z": true,  // doy 1
		"2026-02-10T00:00:00Z": true,  // doy 41
		"2026-03-22T00:00:00Z": true,  // doy 81
		"2027-01-01T00:00:00Z": true,  // re-aligned on the next Jan 1
		"2026-01-02T00:00:00Z": false, // doy 2
		"2026-12-31T00:00:00Z": false, // doy 365, not a multiple of 40
	})
}

func TestIntervalYearContainerLeapAndCommon(t *testing.T) {
	// doy 41 is Feb 10 in a common year and in a leap year alike (both share the
	// same January length), so the anchor is stable across leap years.
	holdsCase(t, "+[40d]", map[string]bool{
		"2024-02-10T00:00:00Z": true, // 2024 is a leap year
		"2025-02-10T00:00:00Z": true, // 2025 is common
	})
}

// --- degenerate strides ---

func TestIntervalEveryDayHoldsEveryMidnight(t *testing.T) {
	// +[1d] → week container, stride 1: every midnight, any weekday, any year.
	holdsCase(t, "+[1d]", map[string]bool{
		"2026-07-14T00:00:00Z": true,
		"2026-07-15T00:00:00Z": true,
		"2026-07-14T00:00:01Z": false,
	})
	// Extreme year exercises dayOfYear/weekday at a proleptic-calendar edge.
	if !mustParse(t, "+[1d]").Holds(time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("+[1d] holds at year 1, Jan 1 midnight")
	}
}

func TestIntervalEvery60MinutesIsHourly(t *testing.T) {
	// 60 minutes fills the hour container exactly: top of every hour.
	holdsCase(t, "+[60mn]", map[string]bool{
		"2026-07-14T09:00:00Z": true,
		"2026-07-14T10:00:00Z": true,
		"2026-07-14T09:30:00Z": false,
	})
}

func TestIntervalOverAYearReAlignsAnnually(t *testing.T) {
	// A stride longer than a year has no larger civil cycle, so only Jan 1 (the
	// year boundary) can satisfy day-of-year ≡ 0.
	holdsCase(t, "+[400d]", map[string]bool{
		"2026-01-01T00:00:00Z": true,
		"2027-01-01T00:00:00Z": true,
		"2026-06-01T00:00:00Z": false,
	})
}

// --- zone alignment (containers are civil, so no epoch/UTC drift) ---

func TestIntervalAlignsToLocalMidnightNotUTC(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("zone data unavailable: %v", err)
	}
	p := mustParse(t, "+[2h]")
	// 02:00 *local* is on the grid regardless of the zone's UTC offset (-04:00
	// in July); the same wall-clock instant in UTC (06:00) is also on its own
	// grid, but the point is the local field drives it.
	if !p.Holds(time.Date(2026, 7, 14, 2, 0, 0, 0, loc)) {
		t.Fatal("+[2h] should hold at 02:00 local")
	}
	if p.Holds(time.Date(2026, 7, 14, 3, 0, 0, 0, loc)) {
		t.Fatal("+[2h] must miss 03:00 local")
	}
}

// --- composition, derivation, and errors ---

func TestIntervalComposesWithFieldsAndBounds(t *testing.T) {
	p := mustParse(t, "M-F +[90mn] >=6 <=18")
	if !p.Holds(at(t, "2026-07-14T07:30:00Z")) { // Tue, on grid, in window
		t.Fatal("should hold")
	}
	if p.Holds(at(t, "2026-07-18T07:30:00Z")) { // Saturday
		t.Fatal("should miss on the weekend")
	}
	if p.Holds(at(t, "2026-07-14T04:30:00Z")) { // on grid but before the window
		t.Fatal("should miss before the since bound")
	}
}

func TestIntervalDerivation(t *testing.T) {
	got, ok := mustParse(t, "+[90mn]").Next(at(t, "2026-07-14T00:00:00Z"))
	if !ok || !got.Equal(at(t, "2026-07-14T01:30:00Z")) {
		t.Fatalf("Next = %s, %v", got.Format(time.RFC3339), ok)
	}
}

func TestIntervalSecondGrainWildcardsSecondField(t *testing.T) {
	// A second-grained interval owns the second field, so it defaults to wildcard.
	if got := mustParse(t, "+[30s]").Canonical(); got != "*/*/* * *:*:* +[30s]" {
		t.Fatalf("Canonical = %q", got)
	}
}

func TestIntervalErrors(t *testing.T) {
	if _, err := Parse("-[90mn]"); !errors.Is(err, ErrContext) {
		t.Fatalf("descending interval: %v", err)
	}
	if _, err := Parse("+[0mn]"); !errors.Is(err, ErrRange) {
		t.Fatalf("zero interval: %v", err)
	}
	if _, err := Parse("+[99999999999999999999h]"); !errors.Is(err, ErrRange) {
		t.Fatalf("overflow interval: %v", err)
	}
}

func TestBareWeekUnitIsNotAnInterval(t *testing.T) {
	// A bare `+[3w]` uses the week unit (not an interval grain), so it is not
	// extracted as an interval; it falls through to the (invalid) weekday step.
	if _, err := Parse("+[3w] noon"); !errors.Is(err, ErrContext) {
		t.Fatalf("bare week step: %v", err)
	}
}
