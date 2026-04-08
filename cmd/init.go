package cmd

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sync"

	"grpc-agent/proto/gen/proto"
	"grpc-agent/shared"

	"github.com/hashicorp/go-plugin"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

var (
	port       int
	grpcServer *grpc.Server
	clients    = make(map[string]proto.AgentService_ConnectServer)
	mu         sync.RWMutex
	pendingMu  sync.RWMutex
	pending    = make(map[string]chan *proto.AgentMessage)
)

var initCmdObj = &cobra.Command{
	Use:   "init",
	Short: "Start the bi-directional streaming grpc server",
	Run: func(cmd *cobra.Command, args []string) {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		grpcServer = grpc.NewServer()
		proto.RegisterAgentServiceServer(grpcServer, &serverHandler{})

		log.Printf("Server started on port %d", port)

		go func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)
			<-sigCh
			log.Println("Shutting down server...")
			grpcServer.GracefulStop()
		}()

		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	},
}

// serverHandler is the main gRPC server that handles client connections.
// It delegates command execution to plugins.
type serverHandler struct {
	proto.UnimplementedAgentServiceServer
	pluginClient *plugin.Client
	executor     shared.Executor
}

func (s *serverHandler) Connect(stream proto.AgentService_ConnectServer) error {
	clientName := ""
	clientCtx := stream.Context()

	if p, ok := peer.FromContext(clientCtx); ok {
		log.Printf("Client connected from: %s", p.Addr)
	}

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			log.Printf("Client %s disconnected", clientName)
			mu.Lock()
			delete(clients, clientName)
			mu.Unlock()
			return nil
		}
		if err != nil {
			log.Printf("Error receiving from client %s: %v", clientName, err)
			return err
		}

		if clientName == "" {
			clientName = msg.ClientName
			mu.Lock()
			clients[clientName] = stream
			mu.Unlock()
			log.Printf("Client registered: %s", clientName)

			// Initialize the plugin client on first connection
			if s.pluginClient == nil {
				s.initializePluginClient()
			}

			resp := &proto.AgentMessage{
				ClientName: clientName,
				Command:    "",
				Output:     "Connected to server",
				IsResponse: true,
			}
			if err := stream.Send(resp); err != nil {
				log.Printf("Error sending to client %s: %v", clientName, err)
				return err
			}
		}

		if msg.Command != "" && !msg.IsResponse {
			targetName := msg.TargetName
			if targetName == "" {
				targetName = clientName
			}

			log.Printf("Executing command on %s: %s", targetName, msg.Command)

			if targetName == clientName {
				// Execute locally using the plugin
				output, err := s.executor.Execute(stream.Context(), msg.Command)
				resp := &proto.AgentMessage{
					ClientName: clientName,
					Command:    msg.Command,
					Output:     output,
					IsResponse: true,
				}
				if err != nil {
					resp.Output = fmt.Sprintf("Error: %v", err)
				}

				if err := stream.Send(resp); err != nil {
					log.Printf("Error sending response to client %s: %v", clientName, err)
					return err
				}
			} else {
				mu.RLock()
				targetStream, ok := clients[targetName]
				mu.RUnlock()

				if !ok {
					resp := &proto.AgentMessage{
						ClientName: clientName,
						Command:    msg.Command,
						Output:     fmt.Sprintf("client %s not found", targetName),
						IsResponse: true,
					}
					stream.Send(resp)
				} else {
					responseCh := make(chan *proto.AgentMessage, 1)
					pendingMu.Lock()
					pending[clientName] = responseCh
					pendingMu.Unlock()

					forwardMsg := &proto.AgentMessage{
						ClientName: clientName,
						TargetName: targetName,
						Command:    msg.Command,
						IsResponse: false,
					}
					if err := targetStream.Send(forwardMsg); err != nil {
						pendingMu.Lock()
						delete(pending, clientName)
						pendingMu.Unlock()
						stream.Send(&proto.AgentMessage{
							ClientName: clientName,
							Command:    msg.Command,
							Output:     fmt.Sprintf("Error forwarding command: %v", err),
							IsResponse: true,
						})
						continue
					}

					clientResp := <-responseCh
					pendingMu.Lock()
					delete(pending, clientName)
					pendingMu.Unlock()

					resp := &proto.AgentMessage{
						ClientName: clientName,
						Command:    msg.Command,
						Output:     clientResp.Output,
						IsResponse: true,
					}
					stream.Send(resp)
				}
			}
			continue
		}

		if msg.IsResponse && msg.TargetName != "" {
			pendingMu.RLock()
			responseCh, ok := pending[msg.TargetName]
			pendingMu.RUnlock()
			if ok {
				responseCh <- msg
			}
		}
	}
}

// initializePluginClient sets up the plugin client and loads the executor plugin.
func (s *serverHandler) initializePluginClient() {
	// Get the plugin command from environment or use default
	pluginCmd := os.Getenv("EXECUTOR_PLUGIN")
	if pluginCmd == "" {
		pluginCmd = resolveDefaultPluginPath()
	}

	s.pluginClient = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  shared.HandshakeConfig,
		Plugins:          shared.PluginMap,
		Cmd:              exec.Command(pluginCmd),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolNetRPC},
	})

	// Connect via RPC
	rpcClient, err := s.pluginClient.Client()
	if err != nil {
		log.Fatalf("error creating plugin client: %v", err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(shared.PluginExecutor)
	if err != nil {
		log.Fatalf("error dispensing plugin: %v", err)
	}

	// We should have an Executor now
	s.executor = raw.(shared.Executor)
	log.Println("Successfully loaded executor plugin")
}

func resolveDefaultPluginPath() string {
	execPath, err := os.Executable()
	if err != nil {
		return "./plugins/local_exec_plugin/local_exec_plugin"
	}
	return filepath.Join(filepath.Dir(execPath), "plugins", "local_exec_plugin", "local_exec_plugin")
}

func InitCmd() *cobra.Command {
	return initCmdObj
}

func init() {
	initCmdObj.Flags().IntVarP(&port, "port", "p", 50051, "Port to listen on")
}
