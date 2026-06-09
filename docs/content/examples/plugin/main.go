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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	plugin := NewServiceRulesPlugin()

	app, err := server.NewFromConfig(ctx,
		server.WithLogger(logger),
		server.WithPlugin(plugin),
	)
	if err != nil {
		logger.Error("failed to create service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("starting anonymizer with service-rules plugin")
	if err := app.ListenAndServe(ctx); err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
