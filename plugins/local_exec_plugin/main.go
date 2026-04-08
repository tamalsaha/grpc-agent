package main

import (
	"context"
	"os"
	"os/exec"

	"grpc-agent/shared"

	"github.com/hashicorp/go-plugin"
)

// LocalExecutor is a simple executor that runs commands locally.
type LocalExecutor struct{}

// Execute runs a command locally and returns the output.
func (e *LocalExecutor) Execute(ctx context.Context, command string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-lc", command)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			shared.PluginExecutor: &shared.ExecutorPlugin{Impl: &LocalExecutor{}},
		},
	})
}
