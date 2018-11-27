package crossconnect

//go:generate protoc -I . -I ../../../../vendor/ crossconnect.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
