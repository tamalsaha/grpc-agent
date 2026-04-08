package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"grpc-agent/proto/gen/proto"
)

func main() {
	serverAddr := os.Getenv("GRPC_AGENT_SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = "localhost:50051"
	}
	clientName := os.Getenv("GRPC_AGENT_CLIENT_NAME")
	if clientName == "" {
		clientName, _ = os.Hostname()
	}

	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: remote_exec_plugin <client-name> <command> [args...]\n")
		os.Exit(1)
	}

	targetClient := args[0]
	command := ""
	for i, arg := range args[1:] {
		if i > 0 {
			command += " "
		}
		command += arg
	}

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := proto.NewAgentServiceClient(conn)
	stream, err := client.Connect(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}

	err = stream.Send(&proto.AgentMessage{
		ClientName: clientName,
		IsResponse: false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send: %v\n", err)
		os.Exit(1)
	}

	_, err = stream.Recv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to receive: %v\n", err)
		os.Exit(1)
	}

	err = stream.Send(&proto.AgentMessage{
		ClientName: clientName,
		TargetName: targetClient,
		Command:    command,
		IsResponse: false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send command: %v\n", err)
		os.Exit(1)
	}

	resp, err := stream.Recv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to receive response: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(resp.Output)
}
