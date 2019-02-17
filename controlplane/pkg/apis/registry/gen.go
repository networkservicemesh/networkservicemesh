package registry

//go:generate protoc -I . registry.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
