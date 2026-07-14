package isnow

import (
	"errors"
	"io"
	"testing"
)

// parseErr asserts that Parse(src) fails with the given sentinel.
func parseErr(t *testing.T, src string, want error) {
	t.Helper()
	if _, err := Parse(src); !errors.Is(err, want) {
		t.Fatalf("Parse(%q) err = %v, want %v", src, err, want)
	}
}

func TestContextErrors(t *testing.T) {
	cases := []string{
		"/// noon",           // four date slots
		"::: noon",           // four time slots
		"2000// 2001// noon", // two date groups
		"M/1 noon",           // weekday name in the month field
		"M-F/1 noon",         // weekday span in the month field
		"*/*/-2-5 noon",      // from-end with a range
		"6 7",                // two bare numbers, hour claimed twice
		"noon 6:00",          // time symbol beside a time group
		"2 12:00",            // numeric weekday without the three-group form
		"noon >=6,12",        // a set inside a bound
		"noon >=M",           // a weekday inside a bound
		"12 <:0",             // a time bound not anchored at the hour
		"12 <::0",            // a second bound without hour or minute
		"12 >=6::0",          // a second bound with an hour but no minute
		"noon >=*",           // a bound that constrains nothing
		"*-5",                // open-low span
		"-*",                 // from-end of a wildcard
		"-1// noon",          // unbounded year from-end
	}
	for _, src := range cases {
		parseErr(t, src, ErrContext)
	}
}

func TestSymbolErrors(t *testing.T) {
	for _, src := range []string{
		"M,T noon",   // T is ambiguous inside a weekday set
		"MWF-F noon", // a run cannot be a span endpoint
		"*/*/5x noon",
		"mid",
	} {
		parseErr(t, src, ErrSymbol)
	}
}

func TestRangeErrors(t *testing.T) {
	for _, src := range []string{
		"25",            // hour out of range
		"2016-2011// n", // descending year span
		"12 >=25",       // hour bound out of range
		"::+[0]",        // a zero step
	} {
		parseErr(t, src, ErrRange)
	}
}

func TestSyntaxError(t *testing.T) {
	parseErr(t, "M+[3", ErrSyntax)
}

func TestBoundTimeGroup(t *testing.T) {
	// A time group (not a bare number) inside a bound.
	p := mustParse(t, "noon >=6:30")
	if p.Canonical() == "" {
		t.Fatal("empty canonical")
	}
}

func TestYearCycleStepsAreContext(t *testing.T) {
	// Year has no natural cycle: elided and from-end year steps are rejected
	// (the window-as-cycle feature is deferred, decision 004).
	for _, src := range []string{
		"+[4]// noon",             // elided-anchor year step
		"2000-[2]// noon",         // from-end year step
		"-1// noon >=2011 <=2015", // year from-end even when bounded
	} {
		parseErr(t, src, ErrContext)
	}
}

func TestYearArithmeticStepAllowed(t *testing.T) {
	// A numeric-anchor forward year step is an open progression (no cycle).
	if _, err := Parse("2000+[4]// noon"); err != nil {
		t.Fatalf("2000+[4] should parse: %v", err)
	}
}

func TestCodeNilAndUnknown(t *testing.T) {
	if got := Code(nil); got != "" {
		t.Fatalf("Code(nil) = %q, want empty", got)
	}
	if got := Code(io.EOF); got != "" {
		t.Fatalf("Code(io.EOF) = %q, want empty", got)
	}
}

func TestWeekStepNonDayIsContext(t *testing.T) {
	parseErr(t, ":+[3w]", ErrContext) // week unit on the minute field
}

func TestMoreErrorBranches(t *testing.T) {
	cases := []struct {
		src  string
		want error
	}{
		{"25-30", ErrRange},             // span lo out of domain
		{"25-30+[2]", ErrRange},         // span-step with a bad span
		{"8-25", ErrRange},              // span hi out of domain
		{"M-T noon", ErrSymbol},         // ambiguous span endpoint
		{"M,T+[1] noon", ErrSymbol},     // ambiguous weekday in an occurrence step
		{"*/*/M+[3w] noon", ErrContext}, // name anchor on a week step
		{"*/*/1+[0w] noon", ErrRange},   // zero week step
		{":M+[2]", ErrContext},          // name anchor on an arithmetic step
		{":99999+[2]", ErrRange},        // anchor out of domain
		{"8-12+[0]", ErrRange},          // zero step on a span-step
		{"noon >=5x", ErrSymbol},        // bad unit inside a bound
		{"0-A", ErrContext},             // numeric low with a symbolic high
		{"M-5 noon", ErrContext},        // symbolic low with a numeric high
	}
	for _, c := range cases {
		parseErr(t, c.src, c.want)
	}
}

func TestGuardsRejectSilentWrong(t *testing.T) {
	rangeCases := []string{
		":0+[90]",                   // stride >= minute cycle
		"0+[25]",                    // stride >= hour cycle
		"::+[60]",                   // stride == second cycle
		"/-40",                      // tail longer than the day cycle
		"/-0",                       // zero-length tail
		"*/*/* -2w 12:00",           // weekday tail (14 days) exceeds the 7-day cycle
		":0+[99999999999999999999]", // overflow stride
		"/-99999999999999999999",    // overflow tail
		"Monday+[0] noon",           // occurrence index 0
		"Monday+[6] noon",           // occurrence index > 5
		"11/ Th-[6] noon",           // from-end occurrence index > 5
		"/+[99w] noon",              // week stride > 53
		"/5+[3w] noon",              // week anchor >= stride
	}
	for _, src := range rangeCases {
		parseErr(t, src, ErrRange)
	}
}

func TestNumericWeekdaySpanRendersNames(t *testing.T) {
	got := mustParse(t, "*/*/* 2-6 12:00").Canonical()
	if got != "*/*/* Monday-Friday 12:00:00" {
		t.Fatalf("Canonical = %q", got)
	}
	// A numeric weekday step anchor stays numeric (arithmetic, not occurrence).
	if got := mustParse(t, "*/*/* 2+[1] 12:00").Canonical(); got != "*/*/* 2+[1] 12:00:00" {
		t.Fatalf("step anchor Canonical = %q", got)
	}
}

func TestYearStepUnbounded(t *testing.T) {
	// Year steps are open progressions with no cycle guard.
	if got := mustParse(t, "2000+[100]// noon").Canonical(); got != "2000+[100]/*/* * 12:00:00" {
		t.Fatalf("year step Canonical = %q", got)
	}
}

func TestStarAnchorStepCanonical(t *testing.T) {
	// A wildcard step anchor renders as '*'.
	got := mustParse(t, "::*+[2]").Canonical()
	if got != "*/*/* * *:*:*+[2]" {
		t.Fatalf("Canonical = %q", got)
	}
}
