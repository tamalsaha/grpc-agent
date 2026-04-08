# grpc-agent

A gRPC-based agent for remote command execution using bi-directional streaming.

## Installation

```bash
go install github.com/tamalsaha/grpc-agent@latest
```

Or build from source:

```bash
make all
```

## Usage

### Start the Server (init)

Run the gRPC server that clients connect to:

```bash
./grpc-agent init -p 50051
```

### Join as Client (join)

Connect a client to the server:

```bash
./grpc-agent join -s localhost:50051 -n my-client
```

### Execute Command (exec)

Execute commands locally or remotely using plugins:

```bash
# Execute locally
./grpc-agent exec local hostname
./grpc-agent exec local ls -la

# Execute remotely
./grpc-agent exec remote my-client hostname
./grpc-agent exec remote my-client "ls -la /tmp"
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Start the bi-directional streaming gRPC server |
| `join` | Start a gRPC client that listens for commands |
| `exec local` | Execute a command locally using plugin |
| `exec remote` | Execute a command on a remote client via gRPC server |
| `remote_exec` | Execute a command on a remote client (legacy) |

## Plugins

Two plugins are provided in the `plugins/` directory:

- `local_exec_plugin` - Executes commands locally on the host
- `remote_exec_plugin` - Executes commands on remote clients via the gRPC server

Environment variables for remote_exec_plugin:
- `GRPC_AGENT_SERVER_ADDR` - Server address (default: localhost:50051)
- `GRPC_AGENT_CLIENT_NAME` - Client name for identification

## Protocol

The agent uses gRPC bi-directional streaming:

```
Client <--> Server
     <stream>
```

Each `AgentMessage` contains:
- `client_name` - Name of the client
- `target_name` - Target client for command forwarding
- `command` - Command to execute
- `output` - Command output
- `is_response` - Whether this is a response message

## Build

```bash
# Build main binary
make build

# Build plugins
make build-plugins

# Build everything
make all

# Clean
make clean
```

## Generate Proto

```bash
# Using protoc
protoc --go_out=proto/gen --go_opt=paths=source_relative \
       --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
       proto/agent.proto

# Using buf
buf generate
```
