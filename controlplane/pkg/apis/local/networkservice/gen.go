package networkservice

//go:generate protoc -I . networkservice.proto --go_out=plugins=grpc:. --proto_path=../../../../../ --proto_path=$GOPATH/pkg/mod/
