package isnow

// role identifies which of the seven fields a spec constrains.
type role int

const (
	roleYear role = iota
	roleMonth
	roleDay
	roleWeekday
	roleHour
	roleMinute
	roleSecond
	numRoles
)

// instantCtx is a broken-down instant plus the derived context field matchers
// need (cycle lengths, weekday occurrence indices).
type instantCtx struct {
	year, month, day       int
	hour, minute, second   int
	weekday                int // 1=Sunday .. 7=Saturday
	daysInMonth, dayOfYear int
	occ, occFromEnd        int // this day is the occ-th <weekday> of its month
}

// value returns the field value for a role.
func (c instantCtx) value(r role) int {
	switch r {
	case roleYear:
		return c.year
	case roleMonth:
		return c.month
	case roleDay:
		return c.day
	case roleWeekday:
		return c.weekday
	case roleHour:
		return c.hour
	case roleMinute:
		return c.minute
	default:
		return c.second
	}
}

// cycle is the inclusive [lo, hi] range of a role's parent cycle for this
// instant. Year is never a caller: it has no natural cycle, so year from-end and
// elided year steps are rejected at compile time.
func (c instantCtx) cycle(r role) (int, int) {
	switch r {
	case roleMonth:
		return 1, 12
	case roleDay:
		return 1, c.daysInMonth
	case roleWeekday:
		return 1, 7
	case roleHour:
		return 0, 23
	default:
		return 0, 59
	}
}

// pred is a compiled field-term predicate.
type pred func(instantCtx) bool

// fieldSpec is a compiled field: OR of its term predicates, optionally excluded.
type fieldSpec struct {
	text    string
	terms   []pred
	exclude bool
}

// holds reports whether the field constrains-matches the instant.
func (f fieldSpec) holds(c instantCtx) bool {
	m := anyPred(f.terms, c)
	if f.exclude {
		return !m
	}
	return m
}

func anyPred(ps []pred, c instantCtx) bool {
	for _, p := range ps {
		if p(c) {
			return true
		}
	}
	return false
}

// wildcardField is the default for an absent or empty slot.
func wildcardField() fieldSpec {
	return fieldSpec{terms: []pred{func(instantCtx) bool { return true }}, text: "*"}
}
