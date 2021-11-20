# Test P2P Network

Experiment with P2P networks using gRPC in golang.

Packages needed to get protoc and proto gen working

```sh
go get -u github.com/golang/protobuf/proto
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

To generate proto services

```sh
protoc -I ./ --go_out=./gen --go-grpc_out=./gen ./proto/ping-service.proto
```
