// Package domain holds the pure command logic over the isnow library; it takes
// its clock and effects injected so every path is testable.
package domain

import (
	"context"
	"time"

	isnow "github.com/uplang/isnow.go"
)

// Verdict is the outcome of the membership test.
type Verdict struct {
	Canonical string
	Explain   string
	Holds     bool
}

// Query parses src and tests whether it holds at the instant.
func Query(src string, at time.Time) (Verdict, error) {
	p, err := isnow.Parse(src)
	if err != nil {
		return Verdict{}, err
	}
	return Verdict{Holds: p.Holds(at), Canonical: p.Canonical(), Explain: p.Explain()}, nil
}

// Derive returns up to n occurrences after (forward) or before from. It stops
// and returns ctx's error if the context is cancelled, so a caller can bound an
// unbounded search on a pathological pattern.
func Derive(ctx context.Context, src string, from time.Time, n int, forward bool) ([]time.Time, error) {
	p, err := isnow.Parse(src)
	if err != nil {
		return nil, err
	}
	out := make([]time.Time, 0, n)
	cur := from
	for len(out) < n {
		next, ok, err := advance(ctx, p, cur, forward)
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		out = append(out, next)
		cur = next
	}
	return out, nil
}

func advance(ctx context.Context, p isnow.Pattern, from time.Time, forward bool) (time.Time, bool, error) {
	if forward {
		return p.NextContext(ctx, from)
	}
	return p.PrevContext(ctx, from)
}
