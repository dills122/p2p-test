# Test P2P Network

Experiment with P2P networks using gRPC in golang.

Generating proto files

```bash
protoc --go_out=plugins=grpc:. --go_opt=paths=source_relative pkg/ping/ping.proto
```

To run the test file

```sh
go run .
```
