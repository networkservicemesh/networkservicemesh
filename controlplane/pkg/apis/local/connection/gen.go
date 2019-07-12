package connection

//go:generate protoc -I . connection.proto --go_out=plugins=grpc:. --proto_path=../../../../../ --proto_path=$GOPATH/pkg/mod/
