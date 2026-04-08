package shared

import (
	"context"
	"errors"
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// Executor is the interface that we're exposing as a plugin.
type Executor interface {
	Execute(ctx context.Context, command string) (string, error)
}

// Handshake is a common handshake that is shared by plugin and host.
var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "GRPC_AGENT_PLUGIN",
	MagicCookieValue: "simpleagent",
}

const PluginExecutor = "executor"

var PluginMap = map[string]plugin.Plugin{
	PluginExecutor: &ExecutorPlugin{},
}

// ExecutorPlugin exposes Executor over go-plugin net/rpc.
type ExecutorPlugin struct {
	Impl Executor
}

func (p *ExecutorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &RPCServer{Impl: p.Impl}, nil
}

func (p *ExecutorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCClient{client: c}, nil
}

type RPCServer struct {
	Impl Executor
}

type ExecuteArgs struct {
	Command string
}

type ExecuteReply struct {
	Output string
	Error  string
}

func (s *RPCServer) Execute(args *ExecuteArgs, reply *ExecuteReply) error {
	output, err := s.Impl.Execute(context.Background(), args.Command)
	if err != nil {
		reply.Output = ""
		reply.Error = fmt.Sprintf("Error: %v", err)
	} else {
		reply.Output = output
		reply.Error = ""
	}
	return nil
}

// This is the implementation of net/rpc for the executor plugin client side.
type RPCClient struct {
	client *rpc.Client
}

func (g *RPCClient) Execute(ctx context.Context, command string) (string, error) {
	_ = ctx
	args := &ExecuteArgs{Command: command}
	var reply ExecuteReply
	err := g.client.Call("Plugin.Execute", args, &reply)
	if err != nil {
		return "", err
	}
	if reply.Error != "" {
		return "", errors.New(reply.Error)
	}
	return reply.Output, nil
}
