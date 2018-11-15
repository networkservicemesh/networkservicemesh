package networkservice

//go:generate protoc -I . networkservice.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
