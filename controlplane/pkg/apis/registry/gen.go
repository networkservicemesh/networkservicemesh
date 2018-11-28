package registry

//go:generate protoc -I . -I ../../../../vendor/ registry.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
