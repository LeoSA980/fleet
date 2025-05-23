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
	// Чтение конфига
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
		fmt.Print("Enter command: ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		r, err := c.SendCommand(ctx, &pb.CommandRequest{Command: text})
		if err != nil {
			log.Fatalf("could not send command: %v", err)
		}
		fmt.Println("Server response:", r.Output)
		if text == "exit" {
			fmt.Println("Client is shutting down.")
			break
		}
	}
}
