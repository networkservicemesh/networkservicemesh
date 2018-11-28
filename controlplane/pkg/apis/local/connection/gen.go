package connection

//go:generate protoc -I . -I ../../../../../vendor/ connection.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
