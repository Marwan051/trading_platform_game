package clients

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Marwan051/tradding_platform_game/event_listener/internal/db"
	"github.com/Marwan051/tradding_platform_game/event_listener/internal/events"
	glide "github.com/valkey-io/valkey-glide/go/v2"
	"github.com/valkey-io/valkey-glide/go/v2/config"
	"github.com/valkey-io/valkey-glide/go/v2/options"
)

type ValkeyClient struct {
	client     *glide.Client
	streamName string
	lastID     string
	logger     *slog.Logger
	blockTime  time.Duration
	batchSize  int64
	db         db.Database
}

func NewValkeyClient(host string, port int, streamName string, db db.Database, logger *slog.Logger) (*ValkeyClient, error) {
	clientConfig := config.NewClientConfiguration().WithAddress(&config.NodeAddress{
		Host: host,
		Port: port,
	})

	glideClient, err := glide.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey client: %w", err)
	}

	return &ValkeyClient{
		client:     glideClient,
		streamName: streamName,
		lastID:     "$",
		logger:     logger,
		blockTime:  5 * time.Second, // Block for 5s waiting for new events
		batchSize:  100,             // Read up to 100 events per batch
		db:         db,
	}, nil
}

// Stream continuously reads events from the Valkey stream, unmarshals them,
// and calls processEntry for each one. It blocks until ctx is cancelled.
func (vc *ValkeyClient) Stream(ctx context.Context) error {
	vc.logger.Info("starting stream listener",
		slog.String("stream", vc.streamName),
		slog.String("last_id", vc.lastID),
	)

	for {
		select {
		case <-ctx.Done():
			vc.logger.Info("stream listener shutting down")
			return ctx.Err()
		default:
		}

		entries, err := vc.readBatch(ctx)
		if err != nil {
			vc.logger.Error("failed to read from stream",
				slog.String("stream", vc.streamName),
				slog.String("error", err.Error()),
			)
			// Back off before retrying on error
			select {
			case <-time.After(1 * time.Second):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		for _, entry := range entries {
			if err := vc.processEntry(ctx, entry.id, entry.data); err != nil {
				vc.logger.Error("failed to process event",
					slog.String("event_id", entry.id),
					slog.String("error", err.Error()),
				)
				// Continue processing remaining entries
			}
		}
	}
}

type streamEntry struct {
	id   string
	data string
}

func (vc *ValkeyClient) readBatch(ctx context.Context) ([]streamEntry, error) {
	xreadOpts := options.NewXReadOptions().SetBlock(vc.blockTime).SetCount(vc.batchSize)

	result, err := vc.client.XReadWithOptions(ctx, map[string]string{
		vc.streamName: vc.lastID,
	}, *xreadOpts)
	if err != nil {
		return nil, fmt.Errorf("xread failed: %w", err)
	}

	var entries []streamEntry
	streamResp, ok := result[vc.streamName]
	if !ok {
		return entries, nil
	}

	for _, se := range streamResp.Entries {
		var dataValue string
		for _, fv := range se.Fields {
			if fv.Field == "data" {
				dataValue = fv.Value
				break
			}
		}
		if dataValue == "" {
			vc.logger.Warn("stream entry missing 'data' field", slog.String("id", se.ID))
			continue
		}
		entries = append(entries, streamEntry{id: se.ID, data: dataValue})
		vc.lastID = se.ID
	}

	return entries, nil
}

func (vc *ValkeyClient) processEntry(ctx context.Context, entryID string, data string) error {
	baseEvent, payload, err := events.UnmarshalStreamEvent([]byte(data))
	if err != nil {
		return fmt.Errorf("unmarshal failed for event %s: %w", entryID, err)
	}

	vc.logger.Debug("event unmarshalled",
		slog.String("event_id", baseEvent.EventID),
		slog.Int("event_type", int(baseEvent.Type)),
	)

	if err := vc.db.InsertEvent(ctx, baseEvent.EventID, baseEvent.Timestamp, baseEvent.Type, payload); err != nil {
		return fmt.Errorf("db insert failed: %w", err)
	}
	return nil
}

// Close gracefully shuts down the client.
func (vc *ValkeyClient) Close(_ context.Context) error {
	vc.logger.Info("closing valkey client")
	vc.client.Close()
	return nil
}
