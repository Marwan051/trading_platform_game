package service

import (
	"context"
	"log/slog"

	pb "github.com/Marwan051/tradding_platform_game/proto/gen/go/v1/matching_engine"
)

type MatchingEngineService struct {
	pb.UnimplementedMatchingEngineServer
	logger *slog.Logger
}

func NewMatchingEngineService(logger *slog.Logger) *MatchingEngineService {
	return &MatchingEngineService{
		logger: logger,
	}
}

func (s *MatchingEngineService) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		IsHealthy:       true,
		OrdersProcessed: 0,
		UptimeSeconds:   0,
	}, nil
}
