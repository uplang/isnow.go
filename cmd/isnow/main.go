// Command isnow tests, derives, explains, schedules, and serves isnow (DTimpalr)
// date/time patterns. See https://uplang.github.io/docs.isnow.go/.
package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/uplang/isnow.go/internal/app"
	"github.com/uplang/isnow.go/internal/command"
)

var (
	osExit = os.Exit
	args   = os.Args
)

func main() { osExit(run(args)) }

// run builds the command tree with the real environment and maps the result to
// an exit code; it is a thin, testable shim (no os.Exit inside).
func run(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	env := &app.Env{
		Now:   time.Now,
		Out:   os.Stdout,
		Err:   os.Stderr,
		Sleep: app.RealSleep,
		Spawn: app.RealSpawn,
	}
	return command.Report(env.Err, command.Root(env).Run(ctx, args))
}
