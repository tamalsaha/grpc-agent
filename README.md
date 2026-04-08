# grpc-agent

A gRPC-based agent for remote command execution using bi-directional streaming.

## Installation

```bash
go install github.com/tamalsaha/grpc-agent@latest
```

Or build from source:

```bash
go build -o grpc-agent
```

## Usage

### Start the Server (init)

Run the gRPC server that clients connect to:

```bash
grpc-agent init -p 50051
```

### Join as Client (join)

Connect a client to the server:

```bash
grpc-agent join -s localhost:50051 -n my-client
```

### Execute Remote Command (remote_exec)

Execute a command on a remote client:

```bash
grpc-agent remote_exec my-client "hostname"
grpc-agent remote_exec my-client "ls -la /tmp"
```

## Commands

- `init` - Start the bi-directional streaming gRPC server
- `join` - Start a gRPC client that listens for commands from the server
- `remote_exec` - Execute a command on a remote client

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

## Build Proto

```bash
protoc --go_out=proto/gen --go_opt=paths=source_relative \
       --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
       proto/agent.proto
```

Or with buf:

```bash
buf generate
```