package isnow

// stepTerm dispatches a step to week-granular, weekday-occurrence, or arithmetic
// interpretation per the anchor and unit (specs/contracts/semantics.md §Step).
func stepTerm(r role, anchor *rawAtom, in *rawIncr) (pred, error) {
	switch {
	case hasWeekUnit(in):
		return weekStepTerm(r, anchor, in)
	case r == roleWeekday && anchor != nil && anchor.name != "":
		return occurrenceTerm(anchor.name, in)
	default:
		return arithStepTerm(r, anchor, in)
	}
}

func hasWeekUnit(in *rawIncr) bool {
	for _, q := range in.qtys {
		if q.unit == "w" {
			return true
		}
	}
	return false
}

// occurrenceTerm selects the nth <weekday> of the month (or nth from the end).
func occurrenceTerm(name string, in *rawIncr) (pred, error) {
	set, res := resolveWeekday(name)
	if res != resOne {
		return nil, ErrSymbol
	}
	ks := qtyNums(in.qtys)
	if err := occurrenceIndices(ks); err != nil {
		return nil, err
	}
	return func(c instantCtx) bool {
		if !contains(set, c.weekday) {
			return false
		}
		idx := c.occ
		if in.fromEnd {
			idx = c.occFromEnd
		}
		return contains(ks, idx)
	}, nil
}

// occurrenceIndices validates weekday-occurrence selectors: a month holds at
// most five of any weekday, so the index must be 1..5.
func occurrenceIndices(ks []int) error {
	for _, k := range ks {
		if k < 1 || k > 5 {
			return ErrRange
		}
	}
	return nil
}

// weekStepTerm matches days whose zero-based week-of-year index is ≡ anchor mod N.
func weekStepTerm(r role, anchor *rawAtom, in *rawIncr) (pred, error) {
	if r != roleDay {
		return nil, ErrContext
	}
	a, err := anchorNum(anchor)
	if err != nil {
		return nil, err
	}
	n, err := stepN(in)
	if err != nil {
		return nil, err
	}
	if a >= n || n > weeksPerYear {
		return nil, ErrRange // a week stride must be 1..53 and larger than its anchor
	}
	return func(c instantCtx) bool {
		wi := (c.dayOfYear - 1) / 7
		return mod(wi-a, n) == 0
	}, nil
}

// weeksPerYear caps a week-granular step; a year spans at most 53 week buckets.
const weeksPerYear = 53

// arithStepTerm matches an arithmetic progression from the anchor (or the cycle
// edge when the anchor is elided). A '-' step descends from the anchor/cycle end.
func arithStepTerm(r role, anchor *rawAtom, in *rawIncr) (pred, error) {
	a, elided, err := anchorOrElided(r, anchor, in.fromEnd)
	if err != nil {
		return nil, err
	}
	ns, err := stepNs(in)
	if err != nil {
		return nil, err
	}
	if err := boundedStrides(r, ns); err != nil {
		return nil, err
	}
	return func(c instantCtx) bool { return anyStep(c, r, a, elided, in.fromEnd, ns) }, nil
}

// boundedStrides rejects a field-local stride that cannot progress within its
// cycle (stride >= cycle collapses to the anchor). Cross-cycle periods use a
// unit step (e.g. +[90m]) instead. Year steps are open progressions (no cycle).
func boundedStrides(r role, ns []int) error {
	cs := cycleSize(r)
	if cs == 0 {
		return nil
	}
	for _, n := range ns {
		if n >= cs {
			return ErrRange
		}
	}
	return nil
}

func anyStep(c instantCtx, r role, anchor int, elided, fromEnd bool, ns []int) bool {
	base := anchor
	if elided {
		clo, chi := c.cycle(r)
		base = edge(clo, chi, fromEnd)
	}
	v := c.value(r)
	for _, n := range ns {
		if stepHit(v, base, fromEnd, n) {
			return true
		}
	}
	return false
}

func edge(clo, chi int, fromEnd bool) int {
	if fromEnd {
		return chi
	}
	return clo
}

func stepHit(v, base int, fromEnd bool, n int) bool {
	if fromEnd {
		return v <= base && (base-v)%n == 0
	}
	return v >= base && (v-base)%n == 0
}

// anchorOrElided resolves a numeric step anchor. Year steps that would need a
// cycle (an elided or from-end anchor) are rejected: year has no natural cycle
// and the window-as-cycle feature is deferred (decision 004).
func anchorOrElided(r role, anchor *rawAtom, fromEnd bool) (int, bool, error) {
	if anchor == nil || anchor.star {
		return 0, true, yearGuard(r)
	}
	a, err := anchorNum(anchor)
	if err != nil {
		return 0, false, err
	}
	if fromEnd {
		return a, false, yearGuard(r)
	}
	return a, false, nil
}

func yearGuard(r role) error {
	if r == roleYear {
		return ErrContext
	}
	return nil
}

func anchorNum(a *rawAtom) (int, error) {
	if a == nil || a.star {
		return 0, nil
	}
	if a.name != "" {
		return 0, ErrContext
	}
	v := a.qtys[0].num
	if err := inDomain2(v); err != nil {
		return 0, err
	}
	return v, nil
}

// inDomain2 rejects absurd anchors (negatives can't occur; huge values are range).
func inDomain2(v int) error {
	if v < 0 || v > 9999 {
		return ErrRange
	}
	return nil
}

func stepN(in *rawIncr) (int, error) {
	return positive(in.qtys[0].num)
}

func stepNs(in *rawIncr) ([]int, error) {
	out := make([]int, len(in.qtys))
	for i, q := range in.qtys {
		n, err := positive(q.num)
		if err != nil {
			return nil, err
		}
		out[i] = n
	}
	return out, nil
}

func positive(n int) (int, error) {
	if n < 1 {
		return 0, ErrRange
	}
	return n, nil
}

func qtyNums(qs []rawQty) []int {
	out := make([]int, len(qs))
	for i, q := range qs {
		out[i] = q.num
	}
	return out
}

func mod(a, n int) int {
	return ((a % n) + n) % n
}

// spanStepTerm restricts an arithmetic step to an inclusive span.
func spanStepTerm(r role, t rawTerm) (pred, error) {
	sp, err := spanTerm(r, t)
	if err != nil {
		return nil, err
	}
	st, err := arithStepTerm(r, t.lo, t.incr)
	if err != nil {
		return nil, err
	}
	return func(c instantCtx) bool { return sp(c) && st(c) }, nil
}
