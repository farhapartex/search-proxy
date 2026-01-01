package grpc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/farhapartex/search-proxy/internal/config"
	"github.com/farhapartex/search-proxy/internal/handlers"
	pb "github.com/farhapartex/search-proxy/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedSearchServiceServer
	searchHandler *handlers.SearchHandler
	config        *config.Config
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		searchHandler: handlers.NewSearchHandler(cfg),
		config:        cfg,
	}
}

func (s *Server) FederatedSearch(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	if err := s.validateSearchRequest(req); err != nil {
		return nil, err
	}

	log.Printf("Received search request: query=%q, max_results=%d, platforms=%v",
		req.Query, req.MaxResults, req.Platforms)

	searchCtx, cancel := context.WithTimeout(ctx, s.config.Server.ServerTimeout)
	defer cancel()

	response, err := s.searchHandler.Search(searchCtx, req)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("search failed: %v", err))
	}

	return response, nil
}

func (s *Server) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	log.Printf("Health check requested for service: %s", req.Service)

	return &pb.HealthCheckResponse{
		Status:    "healthy",
		Version:   "1.0.0",
		Timestamp: time.Now().Unix(),
	}, nil
}

func (s *Server) validateSearchRequest(req *pb.SearchRequest) error {
	if req.Query == "" {
		return status.Error(codes.InvalidArgument, "query cannot be empty")
	}

	if len(req.Query) > 500 {
		return status.Error(codes.InvalidArgument, "query too long (max 500 characters)")
	}

	if req.MaxResults < 0 {
		return status.Error(codes.InvalidArgument, "max_results cannot be negative")
	}

	if req.MaxResults > 100 {
		return status.Error(codes.InvalidArgument, "max_results cannot exceed 100")
	}

	validPlatforms := map[string]bool{
		"github":        true,
		"stackoverflow": true,
		"reddit":        true,
	}

	for _, platform := range req.Platforms {
		if !validPlatforms[platform] {
			return status.Error(codes.InvalidArgument,
				fmt.Sprintf("invalid platform: %s (valid: github, stackoverflow, reddit)", platform))
		}
	}

	return nil
}
