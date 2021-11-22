# Test P2P Network

Experiment with P2P networks using gRPC in golang.

Packages needed to get protoc and proto gen working

```sh
go get -u google.golang.org/protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

To generate proto services

```sh
protoc --proto_path=proto --go_out=out --go-grpc_out=out --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./proto/ping-service.proto
```

From a Google Tutorial, with minor modifications

```sh
protoc --proto_path=proto --go-grpc_out=out --go-grpc_opt=paths=source_relative ./proto/ping-service.proto
```

To run the test file

```sh
go run .
```
