package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var serverAddr string
var clientName string

var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Execute a command locally or remotely",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(1)
	},
}

var localCmd = &cobra.Command{
	Use:                "local [command...]",
	Short:              "Execute a command locally",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		executeLocal(args)
	},
}

var remoteCmd = &cobra.Command{
	Use:                "remote [client-name] [command...]",
	Short:              "Execute a command on a remote client",
	Args:               cobra.MinimumNArgs(2),
	DisableFlagParsing: true,
	Run: func(cmd *cobra.Command, args []string) {
		executeRemote(args)
	},
}

func executeLocal(args []string) {
	pluginPath := findPlugin("local_exec_plugin")
	if pluginPath == "" {
		fmt.Println("Error: local_exec_plugin not found")
		os.Exit(1)
	}

	cmd := exec.Command(pluginPath, args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: %v\n%s", err, string(output))
		os.Exit(1)
	}
	fmt.Print(string(output))
}

func executeRemote(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: grpc-agent exec remote <client-name> <command> [args...]")
		os.Exit(1)
	}

	pluginPath := findPlugin("remote_exec_plugin")
	if pluginPath == "" {
		fmt.Println("Error: remote_exec_plugin not found")
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

	cmd := exec.Command(pluginPath)
	cmd.Env = append(os.Environ(),
		"GRPC_AGENT_SERVER_ADDR="+serverAddr,
		"GRPC_AGENT_CLIENT_NAME="+clientName,
	)
	cmd.Args = []string{targetClient, command}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error: %v\n%s", err, string(output))
		os.Exit(1)
	}
	fmt.Print(string(output))
}

func findPlugin(name string) string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}

	dir := execPath[:len(execPath)-len("grpc-agent")]

	paths := []string{
		dir + name,
		dir + "plugins/" + name + "/" + name,
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func ExecCmd() *cobra.Command {
	execCmd.AddCommand(localCmd)
	execCmd.AddCommand(remoteCmd)
	return execCmd
}

func init() {
	localCmd.Flags().StringVarP(&serverAddr, "server", "s", "localhost:50051", "Server address")
	localCmd.Flags().StringVarP(&clientName, "name", "n", getHostname(), "Client name")
	remoteCmd.Flags().StringVarP(&serverAddr, "server", "s", "localhost:50051", "Server address")
	remoteCmd.Flags().StringVarP(&clientName, "name", "n", getHostname(), "Client name")
}
