package command

import (
	"context"
	"fmt"
	"io"

	docs "github.com/urfave/cli-docs/v3"
	"github.com/urfave/cli/v3"

	"github.com/tsvsheet/isnow.go/internal/constants"
)

// manRenderer is indirected so the roff renderer's failure path is honestly
// coverable; real runs use urfave's generator over the live command tree.
// Section 1: user commands (the generator's default is 8, system admin).
var manRenderer = func(cmd *cli.Command) (string, error) {
	return docs.ToManWithSection(cmd, 1)
}

// runMan renders the whole CLI's man page (roff) to w. Packaging writes it to
// isnow.1; a human can read it directly with `isnow man | man -l -`.
func runMan(w io.Writer, root *cli.Command) error {
	page, err := manRenderer(root)
	if err != nil {
		return constants.ErrManPage.With(err)
	}
	if _, err := fmt.Fprint(w, page); err != nil {
		return constants.ErrManPage.With(err)
	}
	return nil
}

// manCommand builds the `man` command: it prints the CLI's man page in roff
// form, generated from the same command tree that serves --help.
func manCommand() *cli.Command {
	return &cli.Command{
		Name:  "man",
		Usage: "print the man page (roff) to stdout",
		Action: func(_ context.Context, c *cli.Command) error {
			return runMan(c.Root().Writer, c.Root())
		},
	}
}
