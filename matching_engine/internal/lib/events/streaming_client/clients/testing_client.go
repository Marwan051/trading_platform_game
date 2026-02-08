package clients

import (
	"context"

	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/types"
)

type TestStreamingClient struct {
}

func (*TestStreamingClient) Publish(ctx context.Context, eventData any, eventType types.EventType) error {
	return nil
}

func (*TestStreamingClient) Close(ctx context.Context) error {
	return nil
}
