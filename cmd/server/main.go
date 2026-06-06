package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	app, err := server.NewFromConfig(ctx)
	if err != nil {
		return err
	}

	return app.ListenAndServe(ctx)
}
