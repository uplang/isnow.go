package domain

import (
	"context"
	"time"

	isnow "github.com/tsvsheet/go-isnow"

	"github.com/tsvsheet/isnow.go/internal/app"
	"github.com/tsvsheet/isnow.go/internal/constants"
)

// Wait blocks until src's next occurrence after now, or fails with ErrTimeout if
// timeout elapses first (timeout <= 0 waits indefinitely).
func Wait(ctx context.Context, env *app.Env, src string, timeout time.Duration) error {
	p, err := isnow.Parse(isnow.PatternText(src))
	if err != nil {
		return err
	}
	now := env.Now()
	next, ok := p.Next(now)
	if !ok {
		return constants.ErrNoOccurrence
	}
	return sleepUntil(ctx, env, next.Sub(now), timeout)
}

func sleepUntil(ctx context.Context, env *app.Env, until, timeout time.Duration) error {
	if timeout > 0 && until > timeout {
		if err := env.Sleep(ctx, timeout); err != nil {
			return err
		}
		return constants.ErrTimeout
	}
	return env.Sleep(ctx, until)
}
