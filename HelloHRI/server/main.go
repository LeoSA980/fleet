package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	pb "HelloHRI/proto"

	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Grpc struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"grpc"`
}

type server struct {
	pb.UnimplementedFleetServer
}

func (s *server) SendCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	fmt.Println("Command received:", req.Command)
	if req.Command == "exit" {
		fmt.Println("Server is shutting down on command exit.")
		go func() {
			os.Exit(0)
		}()
	}
	return &pb.CommandResponse{Output: "Hello from server: " + req.Command}, nil
}

func main() {
	// Reading config
	configData, err := os.ReadFile("../config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config.yaml: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		log.Fatalf("Error parsing config.yaml: %v", err)
	}
	addr := fmt.Sprintf("%s:%d", cfg.Grpc.Host, cfg.Grpc.Port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFleetServer(s, &server{})
	fmt.Printf("gRPC сервер запущен на %s\n", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
