package command

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/uplang/isnow.go/internal/app"
	"github.com/uplang/isnow.go/internal/server"
)

// serveRun is the server runner, indirected so tests can substitute a stub.
var serveRun = realServe

// serveCommand starts the HTTP server.
func serveCommand(env *app.Env) *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "start the HTTP time server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "addr", Value: ":8601", Usage: "listen address"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			return serveRun(ctx, env, c.String("addr"))
		},
	}
}

// realServe runs the HTTP server until ctx is cancelled, then shuts down.
func realServe(ctx context.Context, env *app.Env, addr string) error {
	srv := &http.Server{
		Addr:        addr,
		Handler:     server.New(env.Now, env.Sleep).Handler(),
		ReadTimeout: 10 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	_, _ = fmt.Fprintf(env.Err, "isnow serving on %s\n", addr)
	return awaitShutdown(ctx, srv, errc)
}

func awaitShutdown(ctx context.Context, srv *http.Server, errc <-chan error) error {
	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
