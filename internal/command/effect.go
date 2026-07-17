package command

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/uplang/isnow.go/internal/app"
	"github.com/uplang/isnow.go/internal/constants"
	"github.com/uplang/isnow.go/internal/domain"
)

// waitCommand blocks until the next occurrence, then exits 0.
func waitCommand(env *app.Env) *cli.Command {
	return &cli.Command{
		Name:      "wait",
		Usage:     "block until the next occurrence",
		ArgsUsage: argIsnow,
		Flags: []cli.Flag{
			&cli.DurationFlag{Name: "timeout", Usage: "give up after this long (0 = never)"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			src, err := firstArg(c)
			if err != nil {
				return err
			}
			return domain.Wait(ctx, env, src, c.Duration("timeout"))
		},
	}
}

// runCommand executes a command (or a nowtab) at each occurrence.
func runCommand(env *app.Env) *cli.Command {
	return &cli.Command{
		Name:      "run",
		Usage:     "run a command at each occurrence (cron-superset)",
		ArgsUsage: "<isnow> -- CMD [ARG…]   |   --tab FILE",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "tab", Usage: "a nowtab file of <isnow> <command> lines"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			entries, err := entriesFor(c)
			if err != nil {
				return err
			}
			return domain.Run(ctx, env, entries)
		},
	}
}

func entriesFor(c *cli.Command) ([]domain.Entry, error) {
	if tab := c.String("tab"); tab != "" {
		data, err := os.ReadFile(tab)
		if err != nil {
			return nil, constants.ErrReadTab.With(err, tab)
		}
		return domain.ParseNowtab(string(data))
	}
	return inlineEntry(c.Args().Slice())
}

func inlineEntry(args []string) ([]domain.Entry, error) {
	if len(args) < 2 {
		return nil, constants.ErrMissingCommand
	}
	entry, err := domain.CompileEntry(args[0], args[1], args[2:])
	if err != nil {
		return nil, err
	}
	return []domain.Entry{entry}, nil
}
