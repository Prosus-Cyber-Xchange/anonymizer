package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"anonymizer-service-v2/pkg/anonymizer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	plugin := NewServiceRulesPlugin()

	app, err := anonymizer.NewFromConfig(ctx,
		anonymizer.WithLogger(logger),
		anonymizer.WithPlugin(plugin),
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
