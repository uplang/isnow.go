package domain

import (
	"strconv"
	"strings"

	isnow "github.com/tsvsheet/go-isnow"
)

// BuildFields is the builder's per-field input (raw field-algebra text; empty
// means wildcard).
type BuildFields struct {
	Year, Month, Day     string
	Weekday              string
	Hour, Minute, Second string
	Since, Until         string
}

// Build composes an isnow from field inputs, validating it via Parse.
func Build(f BuildFields) (Verdict, string, error) {
	src := composeFields(f)
	p, err := isnow.Parse(isnow.PatternText(src))
	if err != nil {
		return Verdict{}, "", err
	}
	return Verdict{Canonical: p.Canonical(), Explain: p.Explain()}, src, nil
}

func composeFields(f BuildFields) string {
	date := field(f.Year) + "/" + field(f.Month) + "/" + field(f.Day)
	src := date + " " + weekdayField(f.Weekday) + timePart(f)
	return src + bound(">=", f.Since) + bound("<=", f.Until)
}

var weekdayNames = []string{"", "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// weekdayField renders a builder weekday input, mapping numeric values (1-7,
// Sunday = 1) to their canonical names so the composed isnow routes them to the
// weekday slot — a bare number would otherwise land in the hour.
func weekdayField(v string) string {
	if v == "" {
		return "*"
	}
	parts := strings.Split(v, ",")
	for i, p := range parts {
		parts[i] = weekdayMember(p)
	}
	return strings.Join(parts, ",")
}

func weekdayMember(m string) string {
	if lo, hi, ok := strings.Cut(m, "-"); ok && lo != "" {
		return weekdayNameOr(lo) + "-" + weekdayNameOr(hi)
	}
	return weekdayNameOr(m)
}

func weekdayNameOr(tok string) string {
	if n, err := strconv.Atoi(tok); err == nil && n >= 1 && n <= 7 {
		return weekdayNames[n]
	}
	return tok
}

// timePart emits the time group only down to the finest field the user set, so
// coarser fields are wildcards and finer ones default to 0 via the ladder.
func timePart(f BuildFields) string {
	vals := []string{f.Hour, f.Minute, f.Second}
	last := finestSet(vals)
	if last < 0 {
		return ""
	}
	parts := make([]string, last+1)
	for i := 0; i <= last; i++ {
		parts[i] = field(vals[i])
	}
	return " " + strings.Join(parts, ":")
}

func finestSet(vals []string) int {
	last := -1
	for i, v := range vals {
		if v != "" {
			last = i
		}
	}
	return last
}

func field(v string) string {
	if v == "" {
		return "*"
	}
	return v
}

func bound(op, v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return " " + op + v
}
