package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "HelloHRI/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Grpc struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"grpc"`
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

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewFleetClient(conn)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("\n--- Command Menu ---")
		fmt.Println("1: Send text to server")
		fmt.Println("2: Start spinner animation")
		fmt.Println("3: Show health card")
		fmt.Println("exit: Exit client")
		fmt.Print("Enter command: ")
		cmd, _ := reader.ReadString('\n')
		cmd = strings.TrimSpace(cmd)

		switch cmd {
		case "1":
			fmt.Print("Enter text: ")
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			resp, err := c.SendText(ctx, &pb.TextRequest{Text: text})
			cancel()
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Server:", resp.Result)
			}
		case "2":
			ctx, cancel := context.WithCancel(context.Background())
			stream, err := c.StartSpinner(ctx, &pb.SpinnerRequest{})
			if err != nil {
				fmt.Println("Error:", err)
				cancel()
				continue
			}
			fmt.Println("Spinner started. Type 'stop' to stop.")
			stopCh := make(chan struct{})
			go func() {
				for {
					input, _ := reader.ReadString('\n')
					if strings.TrimSpace(input) == "stop" {
						c.Stop(context.Background(), &pb.StopRequest{})
						close(stopCh)
						return
					}
				}
			}()
			for {
				resp, err := stream.Recv()
				if err != nil || resp.Finished {
					fmt.Println("[spinner finished]")
					break
				}
				select {
				case <-stopCh:
					break
				default:
				}
			}
			cancel()
		case "3":
			ctx, cancel := context.WithCancel(context.Background())
			stream, err := c.ShowHealth(ctx, &pb.HealthRequest{})
			if err != nil {
				fmt.Println("Error:", err)
				cancel()
				continue
			}
			fmt.Println("Health card started. Type 'stop' to stop.")
			stopCh := make(chan struct{})
			go func() {
				for {
					input, _ := reader.ReadString('\n')
					if strings.TrimSpace(input) == "stop" {
						c.Stop(context.Background(), &pb.StopRequest{})
						close(stopCh)
						return
					}
				}
			}()
			for {
				resp, err := stream.Recv()
				if err != nil {
					fmt.Println("[health card finished]")
					break
				}
				if resp.Health != nil || resp.Connection != nil || resp.RobotAscii != "" {
					// ничего не выводим на клиенте
				}
				select {
				case <-stopCh:
					break
				default:
				}
			}
			cancel()
		case "exit":
			fmt.Println("Sending exit to server...")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			resp, err := c.Exit(ctx, &pb.ExitRequest{})
			cancel()
			if err != nil {
				fmt.Println("Error sending exit to server:", err)
			} else {
				fmt.Println("Server:", resp.Result)
			}
			fmt.Println("Client is shutting down.")
			return
		default:
			fmt.Println("Unknown command.")
		}
	}
}
