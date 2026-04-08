package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"grpc-agent/proto/gen/proto"
)

func RemoteExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote_exec [client-name] [command...]",
		Short: "Execute a command on a remote client",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			targetClient := args[0]

			commandStr := ""
			for i, arg := range args[1:] {
				if i > 0 {
					commandStr += " "
				}
				commandStr += arg
			}

			conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Fatalf("Failed to connect to server: %v", err)
			}
			defer conn.Close()

			client := proto.NewAgentServiceClient(conn)
			stream, err := client.Connect(cmd.Context())
			if err != nil {
				log.Fatalf("Failed to connect: %v", err)
			}

			err = stream.Send(&proto.AgentMessage{
				ClientName: clientName,
				IsResponse: false,
			})
			if err != nil {
				log.Fatalf("Failed to send: %v", err)
			}

			_, err = stream.Recv()
			if err != nil {
				log.Fatalf("Failed to receive: %v", err)
			}

			err = stream.Send(&proto.AgentMessage{
				ClientName: clientName,
				TargetName: targetClient,
				Command:    commandStr,
				IsResponse: false,
			})
			if err != nil {
				log.Fatalf("Failed to send command: %v", err)
			}

			resp, err := stream.Recv()
			if err != nil {
				log.Fatalf("Failed to receive response: %v", err)
			}

			fmt.Print(resp.Output)
		},
	}

	cmd.Flags().StringVarP(&serverAddr, "server", "s", "localhost:50051", "Server address")
	cmd.Flags().StringVarP(&clientName, "name", "n", getHostname(), "Client name")

	return cmd
}
