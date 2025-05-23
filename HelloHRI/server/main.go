package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
	mu         sync.Mutex
	activeTask string // "spinner", "health", ""
	stopCh     chan struct{}
}

func (s *server) SendText(ctx context.Context, req *pb.TextRequest) (*pb.TextResponse, error) {
	fmt.Println("[TEXT]", req.Text)
	return &pb.TextResponse{Result: "Text received: " + req.Text}, nil
}

func (s *server) StartSpinner(req *pb.SpinnerRequest, stream pb.Fleet_StartSpinnerServer) error {
	s.mu.Lock()
	if s.activeTask != "" {
		s.mu.Unlock()
		return fmt.Errorf("Another task is running")
	}
	s.activeTask = "spinner"
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	frames := []string{"|", "/", "-", "\\", "|", "/", "-", "\\"}
	start := time.Now()
	maxDuration := 60 * time.Second
	idx := 0
	fmt.Println("[SPINNER] started")
	for {
		select {
		case <-s.stopCh:
			fmt.Println("[SPINNER] stopped by client")
			stream.Send(&pb.SpinnerResponse{Frame: "[stopped]", Finished: true})
			s.clearTask()
			return nil
		default:
			if time.Since(start) > maxDuration {
				fmt.Println("[SPINNER] timeout")
				stream.Send(&pb.SpinnerResponse{Frame: "[timeout]", Finished: true})
				s.clearTask()
				return nil
			}
			fmt.Printf("\r[SPINNER] %s", frames[idx%len(frames)])
			stream.Send(&pb.SpinnerResponse{Frame: frames[idx%len(frames)], Finished: false})
			time.Sleep(200 * time.Millisecond)
			idx++
		}
	}
}

func (s *server) ShowHealth(req *pb.HealthRequest, stream pb.Fleet_ShowHealthServer) error {
	s.mu.Lock()
	if s.activeTask != "" {
		s.mu.Unlock()
		return fmt.Errorf("Another task is running")
	}
	s.activeTask = "health"
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	start := time.Now()
	maxDuration := 60 * time.Second
	fmt.Println("[HEALTH] started")
	for {
		select {
		case <-s.stopCh:
			fmt.Println("[HEALTH] stopped by client")
			stream.Send(&pb.HealthCard{RobotAscii: "[stopped]"})
			s.clearTask()
			return nil
		default:
			if time.Since(start) > maxDuration {
				fmt.Println("[HEALTH] timeout")
				stream.Send(&pb.HealthCard{RobotAscii: "[timeout]"})
				s.clearTask()
				return nil
			}
			health := mockHealth()
			conn := mockConnection()
			robot := robotASCII()
			fmt.Println("\n--- ROBOT HEALTH ---")
			fmt.Println(robot)
			if health != nil {
				fmt.Printf("Status: %s | CPU: %.1f%% | Mem: %.1f%% | Current: %.1fA | Uptime: %s\n", health.Status, health.Cpu, health.Mem, health.Current, health.Uptime)
			}
			if conn != nil {
				fmt.Printf("RTT: %.0f ms | Jitter: %.0f ms | Connected: %v\n", conn.Rtt, conn.Jitter, conn.Connected)
			}
			stream.Send(&pb.HealthCard{Health: health, Connection: conn, RobotAscii: robot})
			time.Sleep(2 * time.Second)
		}
	}
}

func (s *server) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeTask != "" && s.stopCh != nil {
		close(s.stopCh)
		return &pb.StopResponse{Result: "Stopped"}, nil
	}
	return &pb.StopResponse{Result: "No active task"}, nil
}

func (s *server) Exit(ctx context.Context, req *pb.ExitRequest) (*pb.ExitResponse, error) {
	go func() {
		time.Sleep(200 * time.Millisecond)
		fmt.Println("[EXIT] server shutting down by client request")
		os.Exit(0)
	}()
	return &pb.ExitResponse{Result: "Server is shutting down"}, nil
}

func (s *server) clearTask() {
	s.mu.Lock()
	s.activeTask = ""
	s.stopCh = nil
	s.mu.Unlock()
}

func mockHealth() *pb.HealthStatus {
	statuses := []string{"ok", "warn", "error"}
	return &pb.HealthStatus{
		Status:  statuses[rand.Intn(len(statuses))],
		Cpu:     20 + rand.Float32()*60, // 20-80%
		Mem:     30 + rand.Float32()*60, // 30-90%
		Current: 8 + rand.Float32()*4,   // 8-12A
		Uptime:  fmt.Sprintf("%dh %dm %ds", rand.Intn(2), rand.Intn(60), rand.Intn(60)),
	}
}

func mockConnection() *pb.ConnectionStatus {
	return &pb.ConnectionStatus{
		Rtt:       40 + rand.Float32()*20, // 40-60ms
		Jitter:    10 + rand.Float32()*20, // 10-30ms
		Connected: true,
	}
}

func robotASCII() string {
	return `
	   [ROBOT]
	   O
	  /|\
	  / \
	[___]
	`
}

func main() {
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
	fmt.Printf("HRI server has started on %s\n", addr)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\nShutting down server...")
		s.GracefulStop()
		os.Exit(0)
	}()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
