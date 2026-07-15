package isnow

// domains are the static per-role value ranges for validation (day pre-clamp).
var domains = [numRoles][2]int{
	roleYear:    {0, 9999},
	roleMonth:   {1, 12},
	roleDay:     {1, 31},
	roleWeekday: {1, 7},
	roleHour:    {0, 23},
	roleMinute:  {0, 59},
	roleSecond:  {0, 59},
}

// compileField compiles a present field into a predicate spec.
func compileField(r role, f rawField) (fieldSpec, error) {
	terms := make([]pred, len(f.terms))
	for i := range f.terms {
		p, err := compileTerm(r, f.terms[i])
		if err != nil {
			return fieldSpec{}, err
		}
		terms[i] = p
	}
	return fieldSpec{exclude: f.exclude, terms: terms}, nil
}

func compileTerm(r role, t rawTerm) (pred, error) {
	switch classifyTerm(t) {
	case kindWildcard:
		return func(instantCtx) bool { return true }, nil
	case kindStep:
		return stepTerm(r, t.lo, t.incr)
	case kindFromEnd:
		return fromEndTerm(r, t)
	case kindSpan:
		return spanTerm(r, t)
	case kindSpanStep:
		if r == roleWeekday {
			return setposTerm(t) // M-F-[1] = last business day (BYSETPOS over a weekday span)
		}
		return spanStepTerm(r, t)
	default:
		return exactTerm(r, t.lo)
	}
}

type termKind int

const (
	kindExact termKind = iota
	kindWildcard
	kindStep
	kindFromEnd
	kindSpan
	kindSpanStep
)

func classifyTerm(t rawTerm) termKind {
	switch {
	case t.lo == nil:
		return kindStep
	case t.incr != nil && t.hi != nil:
		return kindSpanStep
	case t.incr != nil:
		return kindStep
	case t.loFromEnd:
		return kindFromEnd
	case t.hi != nil:
		return kindSpan
	case t.lo.star:
		return kindWildcard
	default:
		return kindExact
	}
}

// exactTerm matches a single value: a weekday set (symbol) or a numeric value.
func exactTerm(r role, a *rawAtom) (pred, error) {
	if a.name != "" {
		return weekdaySetTerm(r, a.name)
	}
	v := a.qtys[0].num
	if err := inDomain(r, v); err != nil {
		return nil, err
	}
	return func(c instantCtx) bool { return c.value(r) == v }, nil
}

func weekdaySetTerm(r role, name string) (pred, error) {
	if r != roleWeekday {
		return nil, ErrContext
	}
	set, res := resolveWeekday(name)
	if res != resOne {
		return nil, ErrSymbol
	}
	return func(c instantCtx) bool { return contains(set, c.weekday) }, nil
}

// spanTerm matches an inclusive range, wrapping on cyclic fields; hi='*' is open
// to the cycle end; a descending year span is out of range.
func spanTerm(r role, t rawTerm) (pred, error) {
	if t.lo.star {
		return nil, ErrContext // *-v has no low bound
	}
	if t.lo.name != "" {
		return weekdaySpanTerm(r, t)
	}
	lo := t.lo.qtys[0].num
	if err := inDomain(r, lo); err != nil {
		return nil, err
	}
	if t.hi.star {
		if r == roleWeekday {
			return nil, ErrContext // weekday spans need concrete endpoints (like symbolic v-*)
		}
		return func(c instantCtx) bool { return c.value(r) >= lo }, nil
	}
	if t.hi.name != "" {
		return nil, ErrContext // a numeric low with a symbolic high
	}
	return boundedSpan(r, lo, t.hi.qtys[0].num)
}

// weekdaySpanTerm matches an inclusive span between two weekday symbols (wrapping).
func weekdaySpanTerm(r role, t rawTerm) (pred, error) {
	if r != roleWeekday || t.hi == nil || t.hi.name == "" {
		return nil, ErrContext
	}
	lo, err := singleWeekday(t.lo.name)
	if err != nil {
		return nil, err
	}
	hi, err := singleWeekday(t.hi.name)
	if err != nil {
		return nil, err
	}
	return boundedSpan(roleWeekday, lo, hi)
}

func singleWeekday(name string) (int, error) {
	set, res := resolveWeekday(name)
	if res != resOne || len(set) != 1 {
		return 0, ErrSymbol
	}
	return set[0], nil
}

func boundedSpan(r role, lo, hi int) (pred, error) {
	if err := inDomain(r, hi); err != nil {
		return nil, err
	}
	if hi >= lo {
		return func(c instantCtx) bool { v := c.value(r); return v >= lo && v <= hi }, nil
	}
	if r == roleYear {
		return nil, ErrRange // years do not wrap
	}
	return func(c instantCtx) bool { v := c.value(r); return v >= lo || v <= hi }, nil
}

// fromEndTerm matches the tail of length L of the parent cycle (L in days for a
// unit compound on the day field). Year from-end needs the window as its cycle,
// deferred to a future version (specs/decisions/004-stepping-scope.md).
func fromEndTerm(r role, t rawTerm) (pred, error) {
	if t.hi != nil || r == roleYear || len(t.lo.qtys) == 0 {
		return nil, ErrContext // from-end needs a numeric magnitude
	}
	length := atomDays(t.lo)
	if length < 1 || length > cycleSize(r) {
		return nil, ErrRange // a tail must be 1..cycle long
	}
	return func(c instantCtx) bool {
		clo, chi := c.cycle(r)
		v := c.value(r)
		return v >= chi-length+1 && v >= clo
	}, nil
}

// cycleSize is the number of distinct values in a role's parent cycle, for
// range-checking tails and step strides. Year has no cycle (0 = unbounded).
func cycleSize(r role) int {
	switch r {
	case roleMonth:
		return 12
	case roleDay:
		return 31
	case roleWeekday:
		return 7
	case roleHour:
		return 24
	case roleMinute, roleSecond:
		return 60
	default:
		return 0
	}
}

// inDomain validates a scalar against a role's static domain.
func inDomain(r role, v int) error {
	d := domains[r]
	if v < d[0] || v > d[1] {
		return ErrRange
	}
	return nil
}

func contains(set []int, v int) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// atomDays is the length in days of a numeric atom, honoring 'w'/'d' units.
func atomDays(a *rawAtom) int {
	total := 0
	for _, q := range a.qtys {
		if q.num < 1 {
			return 0 // an overflow/invalid member makes the whole tail invalid
		}
		total += q.num * unitDays(q.unit)
	}
	return total
}

func unitDays(unit string) int {
	if unit == "w" {
		return 7
	}
	return 1
}
