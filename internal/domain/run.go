package domain

import (
	"context"
	"errors"
	"time"

	isnow "github.com/tsvsheet/go-isnow"

	"github.com/tsvsheet/isnow.go/internal/app"
	"github.com/tsvsheet/isnow.go/internal/constants"
)

// Entry is one nowtab-style schedule: an isnow and the command to run at each
// occurrence.
type Entry struct {
	Command string
	Args    []string
	pattern isnow.Pattern
}

// CompileEntry parses an entry's isnow.
func CompileEntry(src, command string, args []string) (Entry, error) {
	p, err := isnow.Parse(isnow.PatternText(src))
	if err != nil {
		return Entry{}, err
	}
	return Entry{pattern: p, Command: command, Args: args}, nil
}

// Run executes each entry's command at its occurrences until ctx is cancelled.
// Commands run synchronously, so a single entry never overlaps itself.
func Run(ctx context.Context, env *app.Env, entries []Entry) error {
	if len(entries) == 0 {
		return constants.ErrMissingCommand
	}
	for {
		if done, err := runTick(ctx, env, entries); done {
			return err
		}
	}
}

// runTick sleeps until the next occurrence and runs its command, reporting
// whether the scheduler should stop.
func runTick(ctx context.Context, env *app.Env, entries []Entry) (bool, error) {
	when, idx, ok := earliest(entries, env.Now())
	if !ok {
		return true, constants.ErrNoOccurrence
	}
	if err := env.Sleep(ctx, when.Sub(env.Now())); err != nil {
		return true, graceful(err)
	}
	if err := env.Spawn(ctx, entries[idx].Command, entries[idx].Args); err != nil {
		return true, err
	}
	return false, nil
}

// earliest finds the entry whose next occurrence after now is soonest.
func earliest(entries []Entry, now time.Time) (time.Time, int, bool) {
	best, bestIdx, found := time.Time{}, -1, false
	for i, e := range entries {
		next, ok := e.pattern.Next(now)
		if ok && (!found || next.Before(best)) {
			best, bestIdx, found = next, i, true
		}
	}
	return best, bestIdx, found
}

// graceful turns context cancellation into a clean stop.
func graceful(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}
