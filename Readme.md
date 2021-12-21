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
