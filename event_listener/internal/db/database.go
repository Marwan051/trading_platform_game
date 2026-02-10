package db

import (
	"context"
	"time"

	streamtypes "github.com/Marwan051/tradding_platform_game/event_listener/internal/stream_types"
)

type Database interface {
	InsertEvent(ctx context.Context, eventID string, timestamp time.Time, eventType streamtypes.EventType, payload streamtypes.EventPayload) error
}
