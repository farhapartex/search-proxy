package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/farhapartex/search-proxy/internal/config"
	grpcServer "github.com/farhapartex/search-proxy/internal/grpc"
	pb "github.com/farhapartex/search-proxy/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Server will listen on port: %s", cfg.Server.GRPCPort)
	log.Printf("Server timeout: %v", cfg.Server.ServerTimeout)
	log.Printf("Per-API timeout: %v", cfg.Server.PerAPITimeout)

	address := fmt.Sprintf(":%s", cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", address, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.MaxConcurrentStreams(1000),
	)

	searchServer := grpcServer.NewServer(cfg)
	pb.RegisterSearchServiceServer(grpcSrv, searchServer)

	reflection.Register(grpcSrv)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Received shutdown signal, gracefully stopping server...")
		grpcSrv.GracefulStop()
		log.Println("Server stopped")
	}()

	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
