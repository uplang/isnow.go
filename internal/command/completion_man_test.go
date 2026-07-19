package command

import (
	"errors"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/tsvsheet/isnow.go/internal/constants"
)

func TestCompletionEmitsScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		h := newHarness()
		if code := h.run("completion", shell); code != 0 {
			t.Fatalf("completion %s: exit %d, stderr %q", shell, code, h.err.String())
		}
		if !strings.Contains(h.out.String(), "isnow") {
			t.Fatalf("completion %s script does not name the program: %q", shell, h.out.String())
		}
	}
}

func TestCompletionMissingShell(t *testing.T) {
	h := newHarness()
	if code := h.run("completion"); code != 2 {
		t.Fatalf("missing shell: exit %d, want 2 (stderr %q)", code, h.err.String())
	}
	if !strings.Contains(h.err.String(), string(constants.ErrMissingShell)) {
		t.Fatalf("missing shell diagnostic: %q", h.err.String())
	}
}

func TestCompletionUnsupportedShell(t *testing.T) {
	h := newHarness()
	if code := h.run("completion", "powershell"); code != 2 {
		t.Fatalf("unsupported shell: exit %d, want 2 (stderr %q)", code, h.err.String())
	}
	if !strings.Contains(h.err.String(), string(constants.ErrUnsupportedShell)) {
		t.Fatalf("unsupported shell diagnostic: %q", h.err.String())
	}
}

func TestManEmitsRoff(t *testing.T) {
	h := newHarness()
	if code := h.run("man"); code != 0 {
		t.Fatalf("man: exit %d, stderr %q", code, h.err.String())
	}
	out := h.out.String()
	if !strings.Contains(out, ".TH isnow 1") || !strings.Contains(out, "completion") {
		t.Fatalf("man output missing title or subcommands: %q", out[:min(len(out), 200)])
	}
}

func TestManRendererError(t *testing.T) {
	prev := manRenderer
	manRenderer = func(*cli.Command) (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { manRenderer = prev })

	h := newHarness()
	if code := h.run("man"); code != 3 {
		t.Fatalf("renderer failure: exit %d, want 3 (stderr %q)", code, h.err.String())
	}
	if !strings.Contains(h.err.String(), string(constants.ErrManPage)) {
		t.Fatalf("renderer failure diagnostic: %q", h.err.String())
	}
}

// failWriter always errors, exercising the man write-failure path.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestManWriteError(t *testing.T) {
	if err := runMan(failWriter{}, Root(newHarness().env)); !errors.Is(err, constants.ErrManPage) {
		t.Fatalf("write failure: %v, want ErrManPage", err)
	}
}
