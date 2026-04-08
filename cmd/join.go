package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"grpc-agent/proto/gen/proto"
)

var (
	serverAddr string
	clientName string
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
					output := executeLocalCommand(msg.Command)
					resp := &proto.AgentMessage{
						ClientName: clientName,
						Command:    msg.Command,
						Output:     output,
						IsResponse: true,
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

func executeLocalCommand(cmd string) string {
	parts, err := shlex(cmd)
	if err != nil {
		return fmt.Sprintf("Error parsing command: %v", err)
	}

	c := exec.Command(parts[0], parts[1:]...)
	c.Env = os.Environ()
	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\n%s", err, string(output))
	}
	return string(output)
}

func shlex(cmd string) ([]string, error) {
	return splitWords(cmd), nil
}

func splitWords(s string) []string {
	var words []string
	var current []rune
	inSingleQuote := false
	inDoubleQuote := false

	for _, r := range s {
		switch r {
		case '\'':
			if !inSingleQuote {
				inSingleQuote = true
			} else {
				inSingleQuote = false
			}
			current = append(current, r)
		case '"':
			if !inDoubleQuote {
				inDoubleQuote = true
			} else {
				inDoubleQuote = false
			}
			current = append(current, r)
		case ' ':
			if inSingleQuote || inDoubleQuote {
				current = append(current, r)
			} else if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		default:
			current = append(current, r)
		}
	}

	if len(current) > 0 {
		words = append(words, string(current))
	}

	if len(words) == 0 {
		return []string{"sh", "-c", s}
	}
	return words
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
