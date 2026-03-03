package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nojyerac/hermes/internal/bridge"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := bridge.Config{
		Addr: os.Getenv("HERMES_ADDR"),
	}

	// HERMES_ALLOWED_COMMANDS is a comma-separated list of executable names the
	// bridge is permitted to run, e.g. "go,ginkgo,golangci-lint".
	// Defaults to bridge.DefaultAllowedCommands when unset or empty.
	if raw := strings.TrimSpace(os.Getenv("HERMES_ALLOWED_COMMANDS")); raw != "" {
		parts := strings.Split(raw, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		cfg.AllowedCommands = parts
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := bridge.New(cfg, logger)

	if err := srv.ListenAndServe(ctx); err != nil {
		logger.Error("hermes bridge exited with error", slog.Any("error", err))
		os.Exit(1)
	}
}
