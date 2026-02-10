package interceptors

import (
	"context"
	"log/slog"
	"runtime/debug"
	"time"
)

// EventHandler represents a function that processes an event from Valkey stream
type EventHandler func(ctx context.Context, eventID string, data map[string]any) error

// EventProcessorMiddleware represents middleware that wraps event processing
type EventProcessorMiddleware func(EventHandler) EventHandler

// Logger logs event processing details
func Logger(logger *slog.Logger) EventProcessorMiddleware {
	return func(next EventHandler) EventHandler {
		return func(ctx context.Context, eventID string, data map[string]any) error {
			start := time.Now()

			err := next(ctx, eventID, data)

			duration := time.Since(start)
			status := "success"
			if err != nil {
				status = "error"
			}

			logger.Info("event processed",
				"event_id", eventID,
				"status", status,
				"duration", duration.String(),
				"error", err,
			)

			return err
		}
	}
}

// Recovery recovers from panics during event processing
func Recovery(logger *slog.Logger) EventProcessorMiddleware {
	return func(next EventHandler) EventHandler {
		return func(ctx context.Context, eventID string, data map[string]interface{}) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic recovered during event processing",
						"error", r,
						"event_id", eventID,
						"stack", string(debug.Stack()),
					)
					err = nil // Don't propagate panic as error, just log it
				}
			}()

			return next(ctx, eventID, data)
		}
	}
}

// Chain combines multiple middlewares into a single middleware
func Chain(middlewares ...EventProcessorMiddleware) EventProcessorMiddleware {
	return func(final EventHandler) EventHandler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
