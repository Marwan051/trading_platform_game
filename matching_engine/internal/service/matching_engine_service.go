package service

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	streamingclient "github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/events/streaming_client"
	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/events/streaming_client/clients"
	matchingengine "github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/matching_engine"
	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/types"
	pb "github.com/Marwan051/tradding_platform_game/proto/gen/go/v1/matching_engine"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MatchingEngineService struct {
	pb.UnimplementedMatchingEngineServer
	logger         *slog.Logger
	engine         *matchingengine.MatchingEngine
	inDegradedMode atomic.Bool
	ctx            context.Context
	cancel         context.CancelFunc
	streamer       streamingclient.StreamingClient
	wg             sync.WaitGroup
}

func NewMatchingEngineService(logger *slog.Logger, valkeyOptions clients.ValkeyOptions) *MatchingEngineService {
	ctx, cancel := context.WithCancel(context.Background())

	valkeyStreamingClient, err := clients.NewValkeyClient(
		valkeyOptions.ValkeyHost, valkeyOptions.ValkeyPort, valkeyOptions.ValkeyStreamName, 10000, valkeyOptions.ValkeyRequestTimeoutMs,
	)
	if err != nil {
		log.Fatalf("Could not connect to event streaming client with error: %s", err)
	}

	svc := &MatchingEngineService{
		logger:   logger,
		engine:   matchingengine.NewMatchingEngine(valkeyStreamingClient),
		ctx:      ctx,
		cancel:   cancel,
		streamer: valkeyStreamingClient,
	}

	// initial probe (short timeout)
	probeCtx, probeCancel := context.WithTimeout(ctx, 2*time.Second)
	ok, _ := svc.engine.IsEventStreamerHealthy(probeCtx)
	probeCancel()
	svc.inDegradedMode.Store(!ok)

	// start background poller
	svc.wg.Add(1)
	go func() {
		defer svc.wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-svc.ctx.Done():
				return
			case <-ticker.C:
				checkCtx, c := context.WithTimeout(context.Background(), 1*time.Second)
				ok, _ := svc.engine.IsEventStreamerHealthy(checkCtx)
				c()
				svc.inDegradedMode.Store(!ok)
			}
		}
	}()

	return svc
}

func (s *MatchingEngineService) Close(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel() // stop poller
	}

	// wait for background poller to finish
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if s.streamer != nil {
			return s.streamer.Close(ctx)
		}
		return nil
	}
}

func (s *MatchingEngineService) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	healthCheck, err := s.engine.IsEventStreamerHealthy(ctx)
	if err != nil {
		s.inDegradedMode.Store(true)
		s.logger.Error(err.Error())
		return &pb.HealthCheckResponse{
			IsHealthy:       false,
			OrdersProcessed: 0,
			UptimeSeconds:   0,
		}, err
	}

	if healthCheck {
		s.inDegradedMode.Store(false)
	} else {
		s.inDegradedMode.Store(true)
	}
	return &pb.HealthCheckResponse{
		IsHealthy:       healthCheck,
		OrdersProcessed: 0,
		UptimeSeconds:   0,
	}, nil
}

func (s *MatchingEngineService) PlaceOrder(ctx context.Context, req *pb.PlaceOrderRequest) (*pb.PlaceOrderResponse, error) {
	if s.inDegradedMode.Load() {
		// try one short health probe before rejecting
		probeCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		ok, _ := s.engine.IsEventStreamerHealthy(probeCtx)
		cancel()
		if ok {
			s.inDegradedMode.Store(false)
		} else {
			return &pb.PlaceOrderResponse{
				Success:      false,
				ErrorMessage: "Service is in degraded mode and can't accept new requests",
				ErrorCode:    2,
			}, errors.New("Engine is in degraded mode")
		}
	}

	orderID := uuid.New().String()

	// Convert protobuf enums (1-indexed) to Go enums (0-indexed)
	var orderSide types.OrderSide
	if req.Side == 1 {
		orderSide = types.Buy
	} else {
		orderSide = types.Sell
	}

	var orderType types.OrderType
	if req.OrderType == 1 {
		orderType = types.MarketOrder
	} else {
		orderType = types.LimitOrder
	}

	order := &types.Order{
		OrderId:          orderID,
		TraderId:         req.TraderId,
		StockTicker:      req.StockTicker,
		OrderType:        orderType,
		OrderSide:        orderSide,
		Quantity:         int64(req.Quantity),
		LimitPrice:       int64(req.LimitPriceCents),
		AvailableBalance: int64(req.AvailableBalanceCents),
		Timestamp:        time.Now(),
	}
	matches, remainingQty, err := s.engine.SubmitOrder(order)
	if err != nil {
		s.logger.Error("Failed to submit order", "error", err, "order_id", orderID)
		return nil, status.Errorf(codes.InvalidArgument, "failed to place order: %v", err)
	}

	return &pb.PlaceOrderResponse{
		Success:               true,
		OrderId:               orderID,
		WasFilledImmediately:  len(matches) > 0,
		FilledQuantity:        int64(req.Quantity) - int64(remainingQty),
		AverageFillPriceCents: 0, // TODO: Calculate average fill price from matches
	}, nil
}

func (s *MatchingEngineService) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	if s.inDegradedMode.Load() {
		// try one short health probe before rejecting
		probeCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		ok, _ := s.engine.IsEventStreamerHealthy(probeCtx)
		cancel()
		if ok {
			s.inDegradedMode.Store(false)
		} else {
			return &pb.CancelOrderResponse{
				Success:      false,
				ErrorMessage: "Service is in degraded mode and can't accept new requests",
			}, errors.New("Engine is in degraded mode")
		}
	}

	// Convert protobuf enum (1-indexed) to Go enum (0-indexed)
	var orderSide types.OrderSide
	if req.Side == 1 {
		orderSide = types.Buy
	} else {
		orderSide = types.Sell
	}

	found, err := s.engine.CancelOrder(req.StockTicker, req.OrderId, orderSide)
	if err != nil {
		s.logger.Error("Failed to cancel order", "error", err, "order_id", req.OrderId)
		return nil, status.Errorf(codes.InvalidArgument, "failed to cancel order: %v", err)
	}

	if !found {
		s.logger.Warn("Order not found for cancellation", "order_id", req.OrderId, "stock", req.StockTicker)
		return nil, status.Errorf(codes.NotFound, "order not found: %s", req.OrderId)
	}

	return &pb.CancelOrderResponse{
		Success: true,
		OrderId: req.OrderId,
	}, nil
}
