package isnow

import "strings"

// Explain returns a deterministic English description of the isnow, built from
// the terminology vocabulary. The wording is implementation-defined (not pinned
// by the conformance corpus).
func (p Pattern) Explain() string { return p.explanation }

func renderExplain(sl slots, bounds []boundSpec) string {
	date := dateClauses(sl)
	bound := boundClauses(bounds)
	clauses := make([]string, 0, 1+len(date)+len(bound))
	clauses = append(clauses, "holds at "+timeClause(sl))
	clauses = append(clauses, date...)
	clauses = append(clauses, bound...)
	return strings.Join(clauses, " ")
}

func timeClause(sl slots) string {
	return fieldText(sl, roleHour) + ":" + fieldText(sl, roleMinute) + ":" + fieldText(sl, roleSecond)
}

func dateClauses(sl slots) []string {
	var out []string
	out = appendClause(out, sl, roleWeekday, "on ")
	out = appendClause(out, sl, roleDay, "on day ")
	out = appendClause(out, sl, roleMonth, "in month ")
	out = appendClause(out, sl, roleYear, "in year ")
	return out
}

func appendClause(out []string, sl slots, r role, prefix string) []string {
	text := fieldText(sl, r)
	if text == "*" {
		return out
	}
	return append(out, prefix+text)
}

func boundClauses(bounds []boundSpec) []string {
	out := make([]string, len(bounds))
	for i, b := range bounds {
		out[i] = boundVerb(b.op) + " " + renderBoundFields(b.fields)
	}
	return out
}

func boundVerb(op boundKind) string {
	switch op {
	case geBound, gtBound:
		return "from"
	default:
		return "until"
	}
}
