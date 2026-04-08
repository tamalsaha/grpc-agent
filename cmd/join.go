package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"grpc-agent/proto/gen/proto"
	"grpc-agent/shared"

	"github.com/hashicorp/go-plugin"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var joinCmdObj = &cobra.Command{
	Use:   "join",
	Short: "Start a grpc client that listens for commands from the server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Failed to connect to server: %v", err)
		}
		defer conn.Close()

		client := proto.NewAgentServiceClient(conn)
		stream, err := client.Connect(ctx)
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}

		log.Printf("Connecting to server as %s...", clientName)

		err = stream.Send(&proto.AgentMessage{
			ClientName: clientName,
			IsResponse: false,
		})
		if err != nil {
			log.Fatalf("Failed to send registration: %v", err)
		}

		resp, err := stream.Recv()
		if err != nil {
			log.Fatalf("Failed to receive response: %v", err)
		}
		log.Printf("Server response: %s", resp.Output)

		// Set up plugin client for local command execution
		pluginClient := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig:  shared.HandshakeConfig,
			Plugins:          shared.PluginMap,
			Cmd:              exec.Command(resolveLocalExecPluginPath()),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
		})
		defer pluginClient.Kill()

		rpcClient, err := pluginClient.Client()
		if err != nil {
			log.Fatalf("error creating plugin client: %v", err)
		}

		raw, err := rpcClient.Dispense(shared.PluginExecutor)
		if err != nil {
			log.Fatalf("error dispensing plugin: %v", err)
		}
		executor := raw.(shared.Executor)

		go func() {
			for {
				msg, err := stream.Recv()
				if err == io.EOF {
					log.Println("Server disconnected")
					return
				}
				if err != nil {
					log.Printf("Error receiving: %v", err)
					return
				}

				if msg.Command != "" && !msg.IsResponse {
					log.Printf("Received command: %s", msg.Command)
					// Execute the command locally using our plugin
					output, err := executor.Execute(context.Background(), msg.Command)
					resp := &proto.AgentMessage{
						ClientName: clientName,
						TargetName: msg.ClientName,
						Command:    msg.Command,
						Output:     output,
						IsResponse: true,
					}
					if err != nil {
						resp.Output = fmt.Sprintf("Error: %v", err)
					}
					stream.Send(resp)
				}
			}
		}()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		log.Println("Disconnected from server")
	},
}

func JoinCmd() *cobra.Command {
	return joinCmdObj
}

func init() {
	joinCmdObj.Flags().StringVarP(&serverAddr, "server", "s", "localhost:50051", "Server address")
	joinCmdObj.Flags().StringVarP(&clientName, "name", "n", getHostname(), "Client name")
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func resolveLocalExecPluginPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return "./plugins/local_exec_plugin/local_exec_plugin"
	}
	return filepath.Join(filepath.Dir(execPath), "plugins", "local_exec_plugin", "local_exec_plugin")
}
