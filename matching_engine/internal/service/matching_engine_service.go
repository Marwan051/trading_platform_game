package service

import (
	"context"
	"log"
	"log/slog"
	"time"

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
	logger *slog.Logger
	engine matchingengine.MatchingEngine
}

func NewMatchingEngineService(logger *slog.Logger, valkeyOptions clients.ValkeyOptions) *MatchingEngineService {
	valkeyStreamingClient, err := clients.NewValkeyClient(
		valkeyOptions.ValkeyHost, valkeyOptions.ValkeyPort, valkeyOptions.ValkeyStreamName, 10000,
	)
	if err != nil {
		log.Fatalf("Could not connect to event streaming client with error: %s", err)
	}
	return &MatchingEngineService{
		logger: logger,
		engine: *matchingengine.NewMatchingEngine(valkeyStreamingClient),
	}
}

func (s *MatchingEngineService) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		IsHealthy:       true,
		OrdersProcessed: 0,
		UptimeSeconds:   0,
	}, nil
}

func (s *MatchingEngineService) PlaceOrder(ctx context.Context, req *pb.PlaceOrderRequest) (*pb.PlaceOrderResponse, error) {
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
		OrderId:     orderID,
		UserId:      req.UserId,
		BotId:       req.BotId,
		StockTicker: req.StockTicker,
		OrderType:   orderType,
		OrderSide:   orderSide,
		Quantity:    int64(req.Quantity),
		LimitPrice:  int64(req.LimitPriceCents),
		Timestamp:   time.Now(),
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
