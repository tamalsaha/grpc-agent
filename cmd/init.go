package cmd

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"grpc-agent/proto/gen/proto"
)

var (
	port       int
	grpcServer *grpc.Server
	clients    = make(map[string]proto.AgentService_ConnectServer)
	mu         sync.RWMutex
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

type serverHandler struct {
	proto.UnimplementedAgentServiceServer
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
				output, err := executeCommand(msg.Command)
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
					forwardMsg := &proto.AgentMessage{
						ClientName: clientName,
						Command:    msg.Command,
						IsResponse: false,
					}
					targetStream.Send(forwardMsg)

					clientResp, err := targetStream.Recv()
					if err != nil {
						stream.Send(&proto.AgentMessage{
							ClientName: clientName,
							Command:    msg.Command,
							Output:     fmt.Sprintf("Error receiving response: %v", err),
							IsResponse: true,
						})
					} else {
						resp := &proto.AgentMessage{
							ClientName: clientName,
							Command:    msg.Command,
							Output:     clientResp.Output,
							IsResponse: true,
						}
						stream.Send(resp)
					}
				}
			}
		}
	}
}

func executeCommand(cmdStr string) (string, error) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

func InitCmd() *cobra.Command {
	return initCmdObj
}

func init() {
	initCmdObj.Flags().IntVarP(&port, "port", "p", 50051, "Port to listen on")
}
