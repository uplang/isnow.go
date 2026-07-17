package command

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/uplang/isnow.go/internal/app"
	"github.com/uplang/isnow.go/internal/domain"
)

// argIsnow is the usage metavariable naming the pattern argument.
const argIsnow = "<isnow>"

// buildFlagNames are the builder's per-field flags.
var buildFlagNames = []string{"year", "month", "day", "weekday", "hour", "minute", "second", "since", "until"}

// buildCommand composes an isnow from field inputs and prints its canonical form.
func buildCommand(env *app.Env) *cli.Command {
	return &cli.Command{
		Name:  "build",
		Usage: "compose an isnow from field values",
		Flags: buildFlags(),
		Action: func(_ context.Context, c *cli.Command) error {
			v, _, err := domain.Build(fieldsFrom(c))
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(env.Out, v.Canonical)
			return nil
		},
	}
}

func buildFlags() []cli.Flag {
	flags := make([]cli.Flag, len(buildFlagNames))
	for i, name := range buildFlagNames {
		flags[i] = &cli.StringFlag{Name: name, Usage: "field-algebra text for " + name}
	}
	return flags
}

func fieldsFrom(c *cli.Command) domain.BuildFields {
	return domain.BuildFields{
		Year: c.String("year"), Month: c.String("month"), Day: c.String("day"),
		Weekday: c.String("weekday"),
		Hour:    c.String("hour"), Minute: c.String("minute"), Second: c.String("second"),
		Since: c.String("since"), Until: c.String("until"),
	}
}
