# AGENTS.md

This file contains instructions for agents working on this codebase.

## Build Commands

```bash
# Build the binary
go build -o grpc-agent

# Run the binary
./grpc-agent [command]

# Run tests (if any)
go test ./...
```

## Proto Generation

```bash
# Using protoc directly
protoc --go_out=proto/gen --go_opt=paths=source_relative \
       --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
       proto/agent.proto

# Using buf
buf generate
```

## Project Structure

- `main.go` - Entry point with cobra root command
- `cmd/init.go` - Server command (bi-directional streaming gRPC server)
- `cmd/join.go` - Client command (connects to server, listens for commands)
- `cmd/remote_exec.go` - Remote execution command
- `proto/agent.proto` - Protocol buffer definitions
- `proto/gen/proto/` - Generated Go code from proto

## Key Files

| File | Purpose |
|------|---------|
| `cmd/init.go` | gRPC server implementation with `Connect` method for bi-directional streaming |
| `cmd/join.go` | gRPC client that executes received commands locally |
| `cmd/remote_exec.go` | Server-side command to forward execution to target client |
| `proto/agent.proto` | Defines `AgentService` with `Connect` streaming RPC |