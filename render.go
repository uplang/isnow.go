package isnow

import (
	"fmt"
	"strings"
)

// renderCanonical renders the fully-qualified `Y/m/d w H:M:S` form, then any
// intervals (they are main-spec groups, so they precede bounds), then bounds.
func renderCanonical(sl slots, intervals []intervalSpec, bounds []boundSpec) string {
	return renderMain(sl) + renderIntervals(intervals) + renderBounds(bounds)
}

// renderMain renders just the `Y/m/d w H:M:S` core (shared with exclusions).
func renderMain(sl slots) string {
	date := join(sl, "/", roleYear, roleMonth, roleDay)
	tod := join(sl, ":", roleHour, roleMinute, roleSecond)
	return date + " " + fieldText(sl, roleWeekday) + " " + tod
}

func join(sl slots, sep string, roles ...role) string {
	parts := make([]string, len(roles))
	for i, r := range roles {
		parts[i] = fieldText(sl, r)
	}
	return strings.Join(parts, sep)
}

func fieldText(sl slots, r role) string {
	f := sl[r]
	if f == nil || !f.present {
		return "*"
	}
	return renderField(r, *f)
}

func renderField(r role, f rawField) string {
	vals := make([]string, 0, len(f.terms))
	for _, t := range f.terms {
		vals = append(vals, renderTerm(r, t)...)
	}
	prefix := ""
	if f.exclude {
		prefix = "!"
	}
	return prefix + strings.Join(vals, ",")
}

func renderTerm(r role, t rawTerm) []string {
	switch classifyTerm(t) {
	case kindWildcard:
		return []string{"*"}
	case kindStep, kindSpanStep:
		return []string{renderStep(r, t)}
	case kindFromEnd:
		return []string{"-" + renderMagnitude(r, t.lo)}
	case kindSpan:
		return []string{renderSpan(r, t)}
	default:
		return renderExact(r, t.lo)
	}
}

func renderExact(r role, a *rawAtom) []string {
	if a.name != "" {
		return weekdayNamesOf(a.name)
	}
	if r == roleWeekday {
		return []string{capitalize(weekdayNames[a.qtys[0].num])}
	}
	return []string{pad(r, a.qtys[0].num)}
}

func renderSpan(r role, t rawTerm) string {
	lo := renderEndpoint(r, t.lo)
	if t.hi.star {
		return lo + "-*"
	}
	return lo + "-" + renderEndpoint(r, t.hi)
}

func renderEndpoint(r role, a *rawAtom) string {
	if a.name != "" {
		return weekdayNamesOf(a.name)[0]
	}
	if r == roleWeekday {
		return capitalize(weekdayNames[a.qtys[0].num])
	}
	return pad(r, a.qtys[0].num)
}

func renderStep(r role, t rawTerm) string {
	anchor := ""
	if t.lo != nil {
		anchor = renderAnchor(r, t)
	}
	return anchor + renderIncr(t.incr)
}

func renderAnchor(r role, t rawTerm) string {
	if t.hi != nil {
		return renderSpan(r, t)
	}
	if t.lo.name != "" {
		return weekdayNamesOf(t.lo.name)[0]
	}
	if t.lo.star {
		return "*"
	}
	// A numeric weekday step anchor stays numeric: it is an arithmetic
	// progression, distinct from a symbolic anchor's occurrence selection.
	return renderQtys(t.lo.qtys) // step anchors render unpadded
}

func renderIncr(in *rawIncr) string {
	sign := "+"
	if in.fromEnd {
		sign = "-"
	}
	return sign + "[" + renderQtys(in.qtys) + "]"
}

func renderQtys(qs []rawQty) string {
	parts := make([]string, len(qs))
	for i, q := range qs {
		parts[i] = fmt.Sprintf("%d%s", q.num, q.unit)
	}
	return strings.Join(parts, ",")
}

// renderMagnitude renders a numeric atom verbatim; a unit compound (2w1d)
// concatenates its parts, unlike a step's comma-separated quantity list.
func renderMagnitude(r role, a *rawAtom) string {
	if len(a.qtys) == 1 && a.qtys[0].unit == "" {
		return pad(r, a.qtys[0].num)
	}
	var b strings.Builder
	for _, q := range a.qtys {
		_, _ = fmt.Fprintf(&b, "%d%s", q.num, q.unit)
	}
	return b.String()
}

func weekdayNamesOf(name string) []string {
	set, _ := resolveWeekday(name)
	out := make([]string, len(set))
	for i, n := range set {
		out[i] = capitalize(weekdayNames[n])
	}
	return out
}

func pad(r role, v int) string {
	if r == roleYear {
		return fmt.Sprintf("%04d", v)
	}
	return fmt.Sprintf("%02d", v)
}

func capitalize(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

func renderBounds(bounds []boundSpec) string {
	var b strings.Builder
	for _, bound := range bounds {
		_, _ = b.WriteString(" " + boundOpText(bound.op) + bound.text)
	}
	return b.String()
}

func boundOpText(op boundKind) string {
	switch op {
	case geBound:
		return ">="
	case gtBound:
		return ">"
	case leBound:
		return "<="
	default:
		return "<"
	}
}

func renderBoundFields(fields []boundField) string {
	date := boundGroup(fields, "/", roleYear, roleMonth, roleDay)
	tod := boundGroup(fields, ":", roleHour, roleMinute, roleSecond)
	return strings.TrimSpace(date + " " + tod)
}

func boundGroup(fields []boundField, sep string, roles ...role) string {
	var parts []string
	for _, r := range roles {
		if v, ok := boundLookup(fields, r); ok {
			parts = append(parts, pad(r, v))
		}
	}
	return strings.Join(parts, sep)
}

func boundLookup(fields []boundField, r role) (int, bool) {
	for _, bf := range fields {
		if bf.role == r {
			return bf.value, true
		}
	}
	return 0, false
}
