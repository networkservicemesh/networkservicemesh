package connection

//go:generate protoc -I . empty.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
//go:generate protoc -I . timestamp.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
