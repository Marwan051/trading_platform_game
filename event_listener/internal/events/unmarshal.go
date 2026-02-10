package events

import (
	"encoding/json"
	"fmt"

	streamtypes "github.com/Marwan051/tradding_platform_game/event_listener/internal/stream_types"
)

// UnmarshalStreamEvent parses a JSON-encoded event and returns the wrapper Event and the specific payload struct
func UnmarshalStreamEvent(data []byte) (*streamtypes.Event, streamtypes.EventPayload, error) {
	var baseEvent streamtypes.Event
	if err := json.Unmarshal(data, &baseEvent); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal base event: %w", err)
	}

	var payload streamtypes.EventPayload
	switch baseEvent.Type {
	case streamtypes.OrderPlaced:
		payload = &streamtypes.OrderPlacedEvent{}
	case streamtypes.OrderCancelled:
		payload = &streamtypes.OrderCancelledEvent{}
	case streamtypes.OrderFilled:
		payload = &streamtypes.OrderFilledEvent{}
	case streamtypes.OrderPartiallyFilled:
		payload = &streamtypes.OrderPartiallyFilledEvent{}
	case streamtypes.OrderRejected:
		payload = &streamtypes.OrderRejectedEvent{}
	case streamtypes.TradeExecuted:
		payload = &streamtypes.TradeExecutedEvent{}
	case streamtypes.PriceChanged:
		payload = &streamtypes.PriceChangedEvent{}
	default:
		return &baseEvent, nil, fmt.Errorf("unknown event type: %d", baseEvent.Type)
	}

	if err := json.Unmarshal(baseEvent.Data, payload); err != nil {
		return &baseEvent, nil, fmt.Errorf("failed to unmarshal payload for type %d: %w", baseEvent.Type, err)
	}

	return &baseEvent, payload, nil
}
