// Package command wires the urfave/cli/v3 command tree over the domain logic.
package command

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/tsvsheet/isnow.go/internal/app"
)

// Root builds the isnow command tree bound to env. Exit-code handling is left to
// the caller (main), so ExitErrHandler is disabled here.
func Root(env *app.Env) *cli.Command {
	root := query(env) // the default command is the membership test
	root.Name = "isnow"
	root.Usage = "match instants against isnow date/time patterns"
	root.Description = "isnow tests, derives, explains, schedules, and serves isnow patterns."
	root.EnableShellCompletion = true
	root.ShellCompletionCommandName = builtinCompletionName
	root.Writer = env.Out
	root.ErrWriter = env.Err
	root.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	root.Commands = []*cli.Command{
		deriveCommand(env, "next", true),
		deriveCommand(env, "prev", false),
		canonCommand(env),
		explainCommand(env),
		waitCommand(env),
		runCommand(env),
		buildCommand(env),
		serveCommand(env),
		completionCommand(),
		manCommand(),
	}
	return root
}
