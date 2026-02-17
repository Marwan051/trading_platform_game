package server

import (
	"context"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/config"
	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/interceptors"
	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/lib/events/streaming_client/clients"
	"github.com/Marwan051/tradding_platform_game/matching_engine/internal/service"
	pb "github.com/Marwan051/tradding_platform_game/proto/gen/go/v1/matching_engine"
)

type Server struct {
	grpcServer        *grpc.Server
	listener          net.Listener
	logger            *slog.Logger
	matchingEngineSVC *service.MatchingEngineService
	cfg               *config.Config
}

func New(cfg *config.Config, logger *slog.Logger) *Server {
	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.Recovery(logger),
			interceptors.Logger(logger),
		),
	)

	// Register services
	matchingService := service.NewMatchingEngineService(logger, clients.ValkeyOptions{
		ValkeyHost:             cfg.ValkeyHost,
		ValkeyPort:             cfg.ValkeyPort,
		ValkeyStreamName:       cfg.ValkeyStreamName,
		ValkeyRequestTimeoutMs: cfg.ValkeyRequestTimeout,
	})
	pb.RegisterMatchingEngineServer(grpcServer, matchingService)

	// Enable reflection for development (grpcurl, grpcui)
	if cfg.Environment == "development" {
		reflection.Register(grpcServer)
	}

	return &Server{
		grpcServer:        grpcServer,
		logger:            logger,
		cfg:               cfg,
		matchingEngineSVC: matchingService,
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.cfg.GRPCAddr)
	if err != nil {
		return err
	}
	s.listener = listener

	return s.grpcServer.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) {
	// First stop accepting new connections and wait for in-flight RPCs to finish
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		s.logger.Warn("forcing server shutdown")
		s.grpcServer.Stop()
	case <-stopped:
		s.logger.Info("gRPC server stopped accepting new connections")
	}

	// Now close matching engine service to drain and shutdown clients
	if s.matchingEngineSVC != nil {
		// Provide a bounded timeout for service Close to avoid blocking shutdown indefinitely
		closeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := s.matchingEngineSVC.Close(closeCtx); err != nil {
			s.logger.Warn("matching engine service close returned error", "err", err)
		}
	}
}
