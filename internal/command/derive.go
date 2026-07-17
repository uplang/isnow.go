package command

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/uplang/isnow.go/internal/app"
	"github.com/uplang/isnow.go/internal/domain"
)

// deriveCommand builds `next` or `prev`: print occurrences, one RFC 3339 per line.
func deriveCommand(env *app.Env, name string, forward bool) *cli.Command {
	var from, tz string
	return &cli.Command{
		Name:      name,
		Usage:     "print the " + name + " occurrences",
		ArgsUsage: argIsnow,
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "count", Aliases: []string{"n"}, Value: 1, Usage: "how many"},
			instantFlag("from", &from),
			tzFlag(&tz),
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return runDerive(ctx, env, c, from, tz, forward)
		},
	}
}

func runDerive(ctx context.Context, env *app.Env, c *cli.Command, from, tz string, forward bool) error {
	src, err := firstArg(c)
	if err != nil {
		return err
	}
	instant, err := resolveInstant(env, from, tz)
	if err != nil {
		return err
	}
	occ, err := domain.Derive(ctx, src, instant, c.Int("count"), forward)
	if err != nil {
		return err
	}
	printInstants(env, occ)
	return nil
}

func printInstants(env *app.Env, occ []time.Time) {
	for _, t := range occ {
		_, _ = fmt.Fprintln(env.Out, t.Format(time.RFC3339))
	}
}
