.PHONY: build clean test proto lint install

# Build the binary
build:
	go build -o grpc-agent

# Clean build artifacts
clean:
	rm -f grpc-agent

# Run tests
test:
	go test ./...

# Generate proto code
proto:
	protoc --go_out=proto/gen --go_opt=paths=source_relative \
	       --go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
	       proto/agent.proto

# Generate proto using buf
proto-buf:
	buf generate

# Run linter
lint:
	golangci-lint run ./...

# Install binary
install: build
	cp grpc-agent $$GOPATH/bin/

# Run the server
run-init:
	./grpc-agent init

# Run a client
run-join:
	./grpc-agent join

# Default target
all: build
