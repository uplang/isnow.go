package command

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/uplang/isnow.go/internal/app"
	"github.com/uplang/isnow.go/internal/domain"
)

// query is the default command: the membership test. Its exit code is the
// answer (0 holds, 1 does not); --explain prints the canonical form and verdict.
func query(env *app.Env) *cli.Command {
	var at, tz string
	var explain bool
	return &cli.Command{
		ArgsUsage: argIsnow,
		Flags: []cli.Flag{
			instantFlag("at", &at),
			tzFlag(&tz),
			&cli.BoolFlag{Name: "explain", Usage: "print the canonical form and verdict"},
		},
		Action: func(_ context.Context, c *cli.Command) error {
			explain = c.Bool("explain")
			return runQuery(env, c, at, tz, explain)
		},
	}
}

func runQuery(env *app.Env, c *cli.Command, at, tz string, explain bool) error {
	src, err := firstArg(c)
	if err != nil {
		return err // Report surfaces this; `isnow -h` / `isnow help` show full usage
	}
	instant, err := resolveInstant(env, at, tz)
	if err != nil {
		return err
	}
	v, err := domain.Query(src, instant)
	if err != nil {
		return err
	}
	if explain {
		_, _ = fmt.Fprintf(env.Out, "%s\n%t\n", v.Canonical, v.Holds)
	}
	if v.Holds {
		return nil
	}
	return ErrNotHolds
}
