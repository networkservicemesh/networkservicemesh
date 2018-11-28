package networkservice

//go:generate protoc -I . -I ../../../../../vendor/ networkservice.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
