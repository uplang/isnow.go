package isnow

// exclusionSpec is a compiled pattern-level exclusion: the pattern does not hold
// at an instant when every field of the exclusion sub-spec holds there. Absent
// time fields default to wildcard, so `! 12/25` excludes all of December 25.
type exclusionSpec struct {
	text   string
	fields [numRoles]fieldSpec
}

func (e exclusionSpec) excludes(c instantCtx) bool {
	for r := role(0); r < numRoles; r++ {
		if !e.fields[r].holds(c) {
			return false
		}
	}
	return true
}

func compileExclusion(groups []rawGroup) (exclusionSpec, error) {
	sl, err := mapGroups(groups, false, true) // timeWild: exclude the whole matching period
	if err != nil {
		return exclusionSpec{}, err
	}
	fields, err := compileAll(sl)
	if err != nil {
		return exclusionSpec{}, err
	}
	return exclusionSpec{fields: fields, text: " ! " + renderMain(sl)}, nil
}

func compileExclusions(raw [][]rawGroup) ([]exclusionSpec, error) {
	out := make([]exclusionSpec, len(raw))
	for i, groups := range raw {
		e, err := compileExclusion(groups)
		if err != nil {
			return nil, err
		}
		out[i] = e
	}
	return out, nil
}

func renderExclusions(es []exclusionSpec) string {
	s := ""
	for _, e := range es {
		s += e.text
	}
	return s
}
