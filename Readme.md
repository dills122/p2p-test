# Test P2P Network

Experiment with P2P networks using gRPC in golang.

## Getting Started

```bash
go get -u
go mod tidy
```

Generating proto files

```bash
protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative pkg/ping/ping.proto
```

Get info on all the available commands

```sh
go run ./main.go --help

```

You can do a test execution with the ping test command, this will setup two nodes that will ping each other.

```sh
go run ./main.go pingTest

```

## Interactive Shell Mode

You can start up nodes in an interactive shell mode to allow you to communicate with the network.

```sh
go run ./main.go start --address 172.0.0.1:9999
# full list of args `cmd\node\start.go` init()
```

Once your node is booted up successfully, the interactive shell will start and you can begin to enter commands.

Currently the only working commands:

- `send` - send a message to the network and to all online nodes
- `exit` - shutdown node and exit network
