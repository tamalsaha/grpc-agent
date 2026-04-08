package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"grpc-agent/proto/gen/proto"
	"grpc-agent/shared"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RemoteExecutor struct{}

func (e *RemoteExecutor) Execute(ctx context.Context, command string) (string, error) {
	serverAddr := getenvOrDefault("GRPC_AGENT_SERVER_ADDR", "localhost:50051")
	clientName := getenvOrDefault("GRPC_AGENT_CLIENT_NAME", hostnameOrUnknown())
	targetClient := os.Getenv("GRPC_AGENT_TARGET_CLIENT")
	if targetClient == "" {
		return "", fmt.Errorf("GRPC_AGENT_TARGET_CLIENT is required for remote mode")
	}

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("failed to connect to server %s: %w", serverAddr, err)
	}
	defer conn.Close()

	client := proto.NewAgentServiceClient(conn)
	stream, err := client.Connect(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to open stream: %w", err)
	}

	if err := stream.Send(&proto.AgentMessage{
		ClientName: clientName,
		IsResponse: false,
	}); err != nil {
		return "", fmt.Errorf("failed to register client: %w", err)
	}

	if _, err := stream.Recv(); err != nil {
		return "", fmt.Errorf("failed to receive registration ack: %w", err)
	}

	if err := stream.Send(&proto.AgentMessage{
		ClientName: clientName,
		TargetName: targetClient,
		Command:    command,
		IsResponse: false,
	}); err != nil {
		return "", fmt.Errorf("failed to send command: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return "", fmt.Errorf("failed to receive command response: %w", err)
	}

	return resp.Output, nil
}

func getenvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hostnameOrUnknown() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func main() {
	log.SetOutput(os.Stderr)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginExecutor: &shared.ExecutorPlugin{Impl: &RemoteExecutor{}},
		},
	})
}
