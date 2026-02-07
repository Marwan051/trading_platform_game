package clients

import (
	"context"
	"sync"

	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/events"
	glide "github.com/valkey-io/valkey-glide/go/v2"
	"github.com/valkey-io/valkey-glide/go/v2/config"
)

type eventPayload struct {
	data      []byte
	eventType events.EventType
}

type ValkeyClient struct {
	client     *glide.Client
	streamName string
	eventChan  chan eventPayload
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewValkeyClient(host string, port int, streamName string, bufferSize int) (*ValkeyClient, error) {
	clientConfig := config.NewClientConfiguration().WithAddress(&config.NodeAddress{
		Host: host,
		Port: port,
	})

	glideClient, err := glide.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	vc := &ValkeyClient{
		client:     glideClient,
		streamName: streamName,
		eventChan:  make(chan eventPayload, bufferSize),
		ctx:        ctx,
		cancel:     cancel,
	}

	vc.wg.Add(1)
	go vc.worker()

	return vc, nil
}
func (vc *ValkeyClient) Publish(ctx context.Context, eventData any, eventType events.EventType) error {

}

func (vc *ValkeyClient) Close(ctx context.Context) error {

}

func (vc *ValkeyClient) worker() {
}
