package isnow

import "fmt"

// An interval is a true periodic recurrence — "every N units" — as distinct from
// a field-local step, which resets within a single field's cycle. An interval is
// anchored hierarchically to the civil calendar (ADR 005): the stride's total
// duration selects the smallest containing civil unit (the container), and the
// interval repeats within each container, re-aligning at the container boundary.
// So the anchor "moves with its unit": `+[2h]` fires at 00,02,…,22 each day and
// re-aligns at every midnight; `+[3d]` fires on Sunday, Wednesday, and Saturday
// and re-aligns every week. Membership stays O(1) — no epoch arithmetic.

// intervalUnit is the grain of an interval step.
type intervalUnit int

const (
	unitSecond intervalUnit = iota
	unitMinute
	unitHour
	unitDay
)

// intervalUnits maps a quantity's unit name to an interval grain; only these
// four fill the cross-cycle gap (weeks/months/years are already fields).
var intervalUnits = map[string]intervalUnit{"s": unitSecond, "mn": unitMinute, "h": unitHour, "d": unitDay}

// seconds is the duration of one grain, in seconds.
func (u intervalUnit) seconds() int {
	switch u {
	case unitSecond:
		return 1
	case unitMinute:
		return 60
	case unitHour:
		return 3600
	default:
		return 86400
	}
}

// onBoundary reports that every field finer than the grain is zero, so the
// instant lies exactly on a grain boundary (a minute grain needs second 0; an
// hour grain minute and second 0; a day grain the whole time 00:00:00).
func (u intervalUnit) onBoundary(c instantCtx) bool {
	switch u {
	case unitSecond:
		return true
	case unitMinute:
		return c.second == 0
	case unitHour:
		return c.minute == 0 && c.second == 0
	default:
		return c.hour == 0 && c.minute == 0 && c.second == 0
	}
}

// minContainer is the smallest civil cycle strictly larger than the grain — the
// floor of the container search (a grain never anchors within its own kind).
func (u intervalUnit) minContainer() container {
	switch u {
	case unitSecond:
		return cMinute
	case unitMinute:
		return cHour
	case unitHour:
		return cDay
	default:
		return cWeek
	}
}

// container is a civil cycle an interval re-aligns to.
type container int

const (
	cMinute container = iota
	cHour
	cDay
	cWeek
	cMonth
	cYear
)

// nominalSeconds is the longest a container of each kind can run. It is used
// only to select the container; the position within it is computed from the
// instant's real fields, so a shorter actual cycle just truncates the tail.
var nominalSeconds = [...]int{
	cMinute: 60,
	cHour:   3600,
	cDay:    86400,
	cWeek:   604800,
	cMonth:  31 * 86400,
	cYear:   366 * 86400,
}

// containerFor picks the smallest civil cycle (at least one grain larger than
// the interval's own grain) whose nominal length holds the whole N-grain stride.
// A stride longer than a year has no larger civil cycle, so it re-aligns
// annually. The comparison divides the nominal by the grain (exact for every
// reachable grain/container pair) to hold the N-stride without risking overflow.
func containerFor(u intervalUnit, n int) container {
	p := u.minContainer()
	for p < cYear && nominalSeconds[p]/u.seconds() < n {
		p++
	}
	return p
}

// secondsInto is the offset, in seconds, from the start of the container to the
// instant. Divided by the grain it gives the grain-position the interval strides
// over; the week container starts on Sunday (weekday 1), matching isnow's own
// weekday numbering.
func secondsInto(p container, c instantCtx) int {
	tod := c.hour*3600 + c.minute*60 + c.second
	switch p {
	case cMinute:
		return c.second
	case cHour:
		return c.minute*60 + c.second
	case cDay:
		return tod
	case cWeek:
		return (c.weekday-1)*86400 + tod
	case cMonth:
		return (c.day-1)*86400 + tod
	default: // cYear
		return (c.dayOfYear-1)*86400 + tod
	}
}

// intervalSpec is a compiled interval: a grain, a stride, and the civil
// container the stride re-aligns to.
type intervalSpec struct {
	text      string
	unit      intervalUnit
	n         int
	container container
}

// holds reports whether the instant lands on the interval grid: it must sit on a
// grain boundary and an integer number of strides into its civil container.
func (iv intervalSpec) holds(c instantCtx) bool {
	if !iv.unit.onBoundary(c) {
		return false
	}
	pos := secondsInto(iv.container, c) / iv.unit.seconds()
	return mod(pos, iv.n) == 0
}

// compileInterval validates and compiles an interval increment.
func compileInterval(in *rawIncr) (intervalSpec, error) {
	if in.fromEnd {
		return intervalSpec{}, ErrContext // a descending interval is meaningless
	}
	q := in.qtys[0]
	n, err := positive(q.num)
	if err != nil {
		return intervalSpec{}, err
	}
	u := intervalUnits[q.unit]
	return intervalSpec{unit: u, n: n, container: containerFor(u, n), text: fmt.Sprintf("+[%d%s]", n, q.unit)}, nil
}

// hasSecondInterval reports whether any interval is second-grained (so the
// second field must default to wildcard rather than 0).
func hasSecondInterval(ins []*rawIncr) bool {
	for _, in := range ins {
		if intervalUnits[in.qtys[0].unit] == unitSecond {
			return true
		}
	}
	return false
}
