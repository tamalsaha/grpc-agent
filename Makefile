.PHONY: build build-plugins clean test integration-test proto lint install

# Build the binary
build:
	go build -o grpc-agent .

# Build plugins
build-plugins:
	go build -o plugins/local_exec_plugin/local_exec_plugin ./plugins/local_exec_plugin/
	go build -o plugins/remote_exec_plugin/remote_exec_plugin ./plugins/remote_exec_plugin/

# Build everything
all: build build-plugins

# Clean build artifacts
clean:
	rm -f grpc-agent plugins/*/*

# Run tests
test:
	go test ./...

# Run end-to-end integration test (server + client + local/remote exec)
integration-test:
	./scripts/integration_test.sh

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
install: build-plugins
	cp grpc-agent $$GOPATH/bin/
	cp plugins/local_exec_plugin/local_exec_plugin $$GOPATH/bin/local_exec_plugin
	cp plugins/remote_exec_plugin/remote_exec_plugin $$GOPATH/bin/remote_exec_plugin

# Run the server
run-init:
	./grpc-agent init

# Run a client
run-join:
	./grpc-agent join

# Run exec local
exec-local:
	./grpc-agent exec local hostname

# Run exec remote
exec-remote:
	./grpc-agent exec remote <client-name> hostname
