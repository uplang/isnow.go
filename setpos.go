package isnow

// setposTerm selects the nth day of the month among those matching a weekday
// span (BYSETPOS over a weekday range): `M-F-[1]` is the last business day,
// `M-F+[1]` the first, `M-F+[1,3]` the first and third. The rank is counted
// among the days in the instant's month whose weekday lies in the span.
func setposTerm(t rawTerm) (pred, error) {
	lo, err := endpointWeekday(t.lo)
	if err != nil {
		return nil, err
	}
	hi, err := endpointWeekday(t.hi)
	if err != nil {
		return nil, err
	}
	set := weekdaySpanSet(lo, hi)
	ks, err := setposIndices(t.incr.qtys)
	if err != nil {
		return nil, err
	}
	fromEnd := t.incr.fromEnd
	return func(c instantCtx) bool {
		return contains(set, c.weekday) && contains(ks, monthRank(c, set, fromEnd))
	}, nil
}

// endpointWeekday resolves a span endpoint (a weekday name or a 1..7 number) to
// a single weekday value.
func endpointWeekday(a *rawAtom) (int, error) {
	if a.name != "" {
		return singleWeekday(a.name)
	}
	if a.star || len(a.qtys) == 0 {
		return 0, ErrContext // a BYSETPOS endpoint must be a concrete weekday
	}
	v := a.qtys[0].num
	if err := inDomain(roleWeekday, v); err != nil {
		return 0, err
	}
	return v, nil
}

// weekdaySpanSet expands an inclusive weekday span, wrapping past Saturday.
func weekdaySpanSet(lo, hi int) []int {
	out := []int{lo}
	for d := lo; d != hi; {
		d = d%7 + 1
		out = append(out, d)
	}
	return out
}

// monthRank returns the 1-based position of the instant's day among the month's
// days matching set, counted from the start (+) or the end (-). The instant's
// own day always lies in [1, daysInMonth], so counting from the edge to it is
// exactly its rank.
func monthRank(c instantCtx, set []int, fromEnd bool) int {
	rank := 0
	if fromEnd {
		for d := c.daysInMonth; d >= c.day; d-- {
			rank += matchDay(c, set, d)
		}
		return rank
	}
	for d := 1; d <= c.day; d++ {
		rank += matchDay(c, set, d)
	}
	return rank
}

func matchDay(c instantCtx, set []int, day int) int {
	if contains(set, weekdayOfDay(c, day)) {
		return 1
	}
	return 0
}

// weekdayOfDay derives the weekday of an arbitrary day of the instant's month
// from the instant's own (day, weekday), by Gregorian arithmetic.
func weekdayOfDay(c instantCtx, day int) int {
	return mod(c.weekday-1+day-c.day, 7) + 1
}

// setposIndices validates BYSETPOS indices (a month has at most 31 days).
func setposIndices(qs []rawQty) ([]int, error) {
	out := make([]int, len(qs))
	for i, q := range qs {
		if q.num < 1 || q.num > 31 {
			return nil, ErrRange
		}
		out[i] = q.num
	}
	return out, nil
}
