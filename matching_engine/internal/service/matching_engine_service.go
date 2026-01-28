package service

import (
	"context"
	"log/slog"
	"time"

	matchingengine "github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/matching_engine"
	pb "github.com/Marwan051/tradding_platform_game/proto/gen/go/v1/matching_engine"
	"github.com/google/uuid"
)

type MatchingEngineService struct {
	pb.UnimplementedMatchingEngineServer
	logger *slog.Logger
	engine matchingengine.MatchingEngine
}

func NewMatchingEngineService(logger *slog.Logger) *MatchingEngineService {
	return &MatchingEngineService{
		logger: logger,
		engine: *matchingengine.NewMatchingEngine(),
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

	order := &matchingengine.Order{
		OrderId:    orderID,
		Stock:      req.StockId,
		OrderType:  matchingengine.OrderType(req.OrderType),
		OrderSide:  matchingengine.OrderSide(req.Side),
		Quantity:   int(req.Quantity),
		LimitPrice: int(req.LimitPriceCents),
		Timestamp:  time.Now(),
	}
	matches, remainingQty, err := s.engine.SubmitOrder(order)
	if err != nil {
		s.logger.Error("Failed to submit order", "error", err, "order_id", orderID)
		return &pb.PlaceOrderResponse{
			Success:      false,
			OrderId:      orderID,
			ErrorMessage: err.Error(),
			ErrorCode:    1, // TODO: Map to proper error codes
		}, nil
	}

	return &pb.PlaceOrderResponse{
		Success:               true,
		OrderId:               orderID,
		WasFilledImmediately:  len(matches) > 0,
		FilledQuantity:        int64(req.Quantity) - int64(remainingQty),
		AverageFillPriceCents: 0, // TODO: Calculate average fill price from matches
	}, nil
}
