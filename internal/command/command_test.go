package command

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	isnow "github.com/tsvsheet/go-isnow"

	"github.com/tsvsheet/isnow.go/internal/app"
	"github.com/tsvsheet/isnow.go/internal/constants"
)

func fixedNow() time.Time { return time.Date(2026, 7, 14, 5, 0, 0, 0, time.UTC) }

type harness struct {
	env *app.Env
	out *bytes.Buffer
	err *bytes.Buffer
}

func newHarness() *harness {
	out, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	return &harness{
		out: out, err: errBuf,
		env: &app.Env{
			Now:   fixedNow,
			Out:   out,
			Err:   errBuf,
			Sleep: func(context.Context, time.Duration) error { return nil },
			Spawn: func(context.Context, string, []string) error { return nil },
		},
	}
}

func (h *harness) run(args ...string) int {
	full := append([]string{"isnow"}, args...)
	// Mirror main: Report prints diagnostics to Err and maps the exit code.
	return Report(h.env.Err, Root(h.env).Run(context.Background(), full))
}

func TestQueryCommand(t *testing.T) {
	h := newHarness()
	if code := h.run("M,W,F noon", "--at", "2026-07-15T12:00:00Z"); code != 0 {
		t.Fatalf("holds exit = %d", code)
	}
	if code := h.run("M,W,F noon", "--at", "2026-07-15T12:00:01Z"); code != 1 {
		t.Fatalf("misses exit = %d", code)
	}
	if code := h.run("25"); code != 2 {
		t.Fatalf("bad isnow exit = %d", code)
	}
	if code := h.run("6", "--at", "bad"); code != 2 {
		t.Fatalf("bad at exit = %d", code)
	}
	if code := h.run("6", "--tz", "Bogus/Zone"); code != 2 {
		t.Fatalf("bad tz exit = %d", code)
	}
}

func TestBareInvocationGuides(t *testing.T) {
	// Bare `isnow` is not silent: it prints guidance and exits 2.
	h := newHarness()
	if code := h.run(); code != 2 {
		t.Fatalf("bare isnow exit = %d, want 2", code)
	}
	if !strings.Contains(h.err.String(), "isnow argument is required") {
		t.Fatalf("bare isnow gave no guidance: out=%q err=%q", h.out.String(), h.err.String())
	}
}

func TestReportPrintsError(t *testing.T) {
	var buf bytes.Buffer
	if code := Report(&buf, isnow.ErrRange); code != 2 {
		t.Fatalf("Report(range) code = %d", code)
	}
	if !strings.Contains(buf.String(), "isnow:") {
		t.Fatalf("Report did not print: %q", buf.String())
	}
	buf.Reset()
	if code := Report(&buf, ErrNotHolds); code != 1 || buf.Len() != 0 {
		t.Fatalf("Report(not-holds) = %d, %q (should be silent)", code, buf.String())
	}
	buf.Reset()
	if code := Report(&buf, nil); code != 0 || buf.Len() != 0 {
		t.Fatalf("Report(nil) = %d, %q", code, buf.String())
	}
}

func TestQueryExplain(t *testing.T) {
	h := newHarness()
	h.run("M,W,F noon", "--at", "2026-07-15T12:00:00Z", "--explain")
	if !strings.Contains(h.out.String(), "true") {
		t.Fatalf("explain out = %q", h.out.String())
	}
}

func TestDeriveCommands(t *testing.T) {
	h := newHarness()
	h.run("next", "6", "--from", "2026-07-14T07:00:00Z", "-n", "2")
	if strings.Count(h.out.String(), "\n") != 2 {
		t.Fatalf("next out = %q", h.out.String())
	}
	if code := h.run("prev", "6", "--from", "2026-07-14T07:00:00Z"); code != 0 {
		t.Fatalf("prev exit = %d", code)
	}
	if code := h.run("next", "25"); code != 2 {
		t.Fatalf("next bad = %d", code)
	}
	if code := h.run("next"); code != 2 {
		t.Fatalf("next no arg = %d", code)
	}
	if code := h.run("next", "6", "--from", "bad"); code != 2 {
		t.Fatalf("next bad from = %d", code)
	}
}

func TestCanonExplainCommands(t *testing.T) {
	h := newHarness()
	if code := h.run("canon", "6"); code != 0 || !strings.Contains(h.out.String(), "06:00:00") {
		t.Fatalf("canon = %d %q", code, h.out.String())
	}
	if code := h.run("canon", "25"); code != 2 {
		t.Fatalf("canon bad = %d", code)
	}
	if code := h.run("canon"); code != 2 {
		t.Fatalf("canon no arg = %d", code)
	}
	h2 := newHarness()
	if code := h2.run("explain", "noon"); code != 0 || !strings.Contains(h2.out.String(), "holds at") {
		t.Fatalf("explain = %d %q", code, h2.out.String())
	}
	if code := h2.run("explain", "25"); code != 2 {
		t.Fatalf("explain bad = %d", code)
	}
	if code := h2.run("explain"); code != 2 {
		t.Fatalf("explain no arg = %d", code)
	}
}

func TestBuildCommand(t *testing.T) {
	h := newHarness()
	if code := h.run("build", "--weekday", "M,W,F", "--hour", "12"); code != 0 {
		t.Fatalf("build = %d", code)
	}
	if !strings.Contains(h.out.String(), "Monday,Wednesday,Friday 12:00:00") {
		t.Fatalf("build out = %q", h.out.String())
	}
	if code := h.run("build", "--hour", "25"); code != 2 {
		t.Fatalf("build bad = %d", code)
	}
}

func TestWaitCommand(t *testing.T) {
	h := newHarness()
	if code := h.run("wait", "*"); code != 0 {
		t.Fatalf("wait = %d", code)
	}
	if code := h.run("wait"); code != 2 {
		t.Fatalf("wait no arg = %d", code)
	}
	// A far occurrence with a short timeout runs to a timeout (exit 3).
	if code := h.run("wait", "6", "--timeout", "1m"); code != 3 {
		t.Fatalf("wait timeout = %d", code)
	}
}

func TestRunCommandInline(t *testing.T) {
	h := newHarness()
	h.env.Sleep = stepSleep(2)
	spawned := 0
	h.env.Spawn = func(context.Context, string, []string) error { spawned++; return nil }
	if code := h.run("run", "*", "echo", "hi"); code != 0 {
		t.Fatalf("run exit = %d", code)
	}
	if spawned != 2 {
		t.Fatalf("spawned %d", spawned)
	}
}

func TestRunCommandErrors(t *testing.T) {
	h := newHarness()
	if code := h.run("run", "6"); code != 2 {
		t.Fatalf("run missing command = %d", code)
	}
	if code := h.run("run", "--tab", filepath.Join(t.TempDir(), "nope")); code != 2 {
		t.Fatalf("run bad tab = %d", code)
	}
}

func TestRunCommandTab(t *testing.T) {
	tab := filepath.Join(t.TempDir(), "nowtab")
	if err := os.WriteFile(tab, []byte("*\techo hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newHarness()
	h.env.Sleep = stepSleep(1)
	if code := h.run("run", "--tab", tab); code != 0 {
		t.Fatalf("run tab = %d", code)
	}
}

func stepSleep(n int) app.Sleeper {
	calls := 0
	return func(context.Context, time.Duration) error {
		calls++
		if calls > n {
			return context.Canceled
		}
		return nil
	}
}

func TestServeCommandStub(t *testing.T) {
	old := serveRun
	defer func() { serveRun = old }()
	called := ""
	serveRun = func(_ context.Context, _ *app.Env, addr string) error {
		called = addr
		return nil
	}
	h := newHarness()
	if code := h.run("serve", "--addr", ":9999"); code != 0 || called != ":9999" {
		t.Fatalf("serve = %d, addr=%q", code, called)
	}
}

func TestRealServeShutdown(t *testing.T) {
	h := newHarness()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- realServe(ctx, h.env, "127.0.0.1:0") }()
	time.Sleep(30 * time.Millisecond)
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("realServe shutdown = %v", err)
	}
}

func TestRealServeBadAddr(t *testing.T) {
	h := newHarness()
	if err := realServe(context.Background(), h.env, "127.0.0.1:99999999"); err == nil {
		t.Fatal("realServe(bad addr) should error")
	}
}

func TestRunInlineBadPattern(t *testing.T) {
	h := newHarness()
	if code := h.run("run", "25", "echo"); code != 2 {
		t.Fatalf("run bad isnow = %d", code)
	}
}

func TestUsageErrorMessage(t *testing.T) {
	if got := (usageError{msg: "need arg"}).Error(); got != "need arg" {
		t.Fatalf("usageError.Error() = %q", got)
	}
}

func TestAwaitShutdownServerClosed(t *testing.T) {
	errc := make(chan error, 1)
	errc <- http.ErrServerClosed
	if err := awaitShutdown(context.Background(), &http.Server{}, errc); err != nil {
		t.Fatalf("awaitShutdown(closed) = %v", err)
	}
}

func TestExitCode(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{ErrNotHolds, 1},
		{isnow.ErrRange, 2},
		{constants.ErrBadTime, 2},
		{usageError{msg: "x"}, 2},
		{constants.ErrTimeout, 3},
		{errors.New("other"), 3},
	}
	for _, c := range cases {
		if got := ExitCode(c.err); got != c.want {
			t.Fatalf("ExitCode(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}
