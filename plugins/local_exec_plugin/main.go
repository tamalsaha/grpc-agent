package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: local_exec_plugin <command> [args...]\n")
		os.Exit(1)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n%s", err, string(output))
		os.Exit(1)
	}
	fmt.Print(string(output))
}
