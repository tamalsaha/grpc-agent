package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/spf13/cobra"
	"grpc-agent/shared"
)

var serverAddr string
var clientName string

var execCmd = &cobra.Command{
	Use:   "exec [local|remote] ...",
	Short: "Execute a command locally or on a remote client via plugins",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("usage: grpc-agent exec local <command...> OR grpc-agent exec remote <client-name> <command...>")
		}

		mode := args[0]
		switch mode {
		case "local":
			return executeWithPlugin(cmd.Context(), "local_exec_plugin", "", args[1:])
		case "remote":
			if len(args) < 3 {
				return fmt.Errorf("usage: grpc-agent exec remote <client-name> <command...>")
			}
			targetClient := args[1]
			return executeWithPlugin(cmd.Context(), "remote_exec_plugin", targetClient, args[2:])
		default:
			return fmt.Errorf("invalid mode %q: use local or remote", mode)
		}
	},
}

func executeWithPlugin(ctx context.Context, pluginName, targetClient string, commandArgs []string) error {
	if len(commandArgs) == 0 {
		return fmt.Errorf("command is required")
	}

	pluginPath := findPlugin(pluginName)
	if pluginPath == "" {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	pluginCmd := exec.Command(pluginPath)
	pluginCmd.Env = append(os.Environ(),
		"GRPC_AGENT_SERVER_ADDR="+serverAddr,
		"GRPC_AGENT_CLIENT_NAME="+clientName,
	)
	if targetClient != "" {
		pluginCmd.Env = append(pluginCmd.Env, "GRPC_AGENT_TARGET_CLIENT="+targetClient)
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.HandshakeConfig,
		Plugins:         shared.PluginMap,
		Cmd:             pluginCmd,
	})
	defer client.Kill()

	rpcClient, err := client.Client()
	if err != nil {
		return fmt.Errorf("failed to start plugin %s: %w", pluginName, err)
	}

	raw, err := rpcClient.Dispense(shared.PluginExecutor)
	if err != nil {
		return fmt.Errorf("failed to dispense plugin %s: %w", pluginName, err)
	}

	executor := raw.(shared.Executor)
	command := strings.Join(commandArgs, " ")
	output, err := executor.Execute(ctx, command)
	if output != "" {
		fmt.Print(output)
	}
	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

func findPlugin(name string) string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}

	execDir := filepath.Dir(execPath)
	paths := []string{
		filepath.Join(execDir, "plugins", name, name),
		filepath.Join(execDir, name),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func ExecCmd() *cobra.Command {
	return execCmd
}

func init() {
	execCmd.Flags().StringVarP(&serverAddr, "server", "s", "localhost:50051", "Server address")
	execCmd.Flags().StringVarP(&clientName, "name", "n", getHostname(), "Client name")
}
