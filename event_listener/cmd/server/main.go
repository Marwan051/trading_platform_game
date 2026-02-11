package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Marwan051/tradding_platform_game/event_listener/internal/config"
	dbpkg "github.com/Marwan051/tradding_platform_game/event_listener/internal/db/postgres"
	dbout "github.com/Marwan051/tradding_platform_game/event_listener/internal/db/postgres/out"
	"github.com/Marwan051/tradding_platform_game/event_listener/internal/events/streaming_client/clients"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	logger.Info("starting event listener service",
		slog.String("environment", cfg.Environment),
	)

	// Setup context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL
	logger.Info("connecting to database", slog.String("url", cfg.DatabaseURL))
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Verify database connection
	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("database connection established")

	// Initialize database layer
	queries := dbout.New(pool)
	database := dbpkg.New(queries)

	// Create Valkey streaming client
	logger.Info("initializing valkey client",
		slog.String("host", cfg.ValkeyHost),
		slog.Int("port", cfg.ValkeyPort),
		slog.String("stream", cfg.ValkeyStreamName),
	)
	valkeyClient, err := clients.NewValkeyClient(
		cfg.ValkeyHost,
		cfg.ValkeyPort,
		cfg.ValkeyStreamName,
		database,
		logger,
	)
	if err != nil {
		logger.Error("failed to create valkey client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := valkeyClient.Close(context.Background()); err != nil {
			logger.Error("failed to close valkey client", slog.String("error", err.Error()))
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start streaming in a goroutine
	errChan := make(chan error, 1)
	go func() {
		logger.Info("starting event stream listener")
		if err := valkeyClient.Stream(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		logger.Info("received shutdown signal", slog.String("signal", sig.String()))
		cancel()
	case err := <-errChan:
		logger.Error("stream error", slog.String("error", err.Error()))
		cancel()
	}

	logger.Info("shutting down gracefully")
}
