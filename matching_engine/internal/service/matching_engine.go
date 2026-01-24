package service

import (
	"context"
	"log/slog"

	pb "github.com/Marwan051/tradding_platform_game/matching_engine/api/proto/v1"
)

type MatchingEngineService struct {
	pb.UnimplementedMatchingEngineServiceServer
	logger *slog.Logger
}

func NewMatchingEngineService(logger *slog.Logger) *MatchingEngineService {
	return &MatchingEngineService{
		logger: logger,
	}
}

func (s *MatchingEngineService) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status: "ok",
	}, nil
}
