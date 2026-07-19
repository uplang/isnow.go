package command

import (
	"context"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/tsvsheet/isnow.go/internal/constants"
)

// builtinCompletionName renames urfave/cli's auto-added (hidden)
// shell-completion command so it does not collide with this repo's own visible
// `completion` command. EnableShellCompletion still drives on-the-fly <TAB>
// completion; the renamed built-in only supplies the per-shell script
// templates the `completion` command delegates to.
const builtinCompletionName = "__completion"

// supportedShells is the ordered set of shells the completion command accepts.
var supportedShells = []string{"bash", "zsh", "fish"}

// supportedShellList renders the supported shells for diagnostics.
func supportedShellList() string { return strings.Join(supportedShells, ", ") }

// completionRenderer resolves the built-in per-shell completion subcommand for
// a supported shell, or nil for an unsupported one.
func completionRenderer(root *cli.Command, shell string) *cli.Command {
	if !slices.Contains(supportedShells, shell) {
		return nil
	}
	return root.Command(builtinCompletionName).Command(shell)
}

// runCompletion writes the completion script for the requested shell to the
// root command's writer, delegating to urfave/cli's built-in templates.
func runCompletion(ctx context.Context, root *cli.Command, shell string) error {
	if shell == "" {
		return constants.ErrMissingShell.With(nil, "supported", supportedShellList())
	}
	renderer := completionRenderer(root, shell)
	if renderer == nil {
		return constants.ErrUnsupportedShell.With(nil, "shell", shell, "supported", supportedShellList())
	}
	return renderer.Action(ctx, renderer)
}

// completionCommand builds the `completion` command: it prints the shell
// completion script for the requested shell so a user can source it.
func completionCommand() *cli.Command {
	return &cli.Command{
		Name:      "completion",
		Usage:     "print a shell completion script for bash, zsh, or fish",
		ArgsUsage: "<shell>",
		Action: func(ctx context.Context, c *cli.Command) error {
			return runCompletion(ctx, c.Root(), c.Args().First())
		},
	}
}
