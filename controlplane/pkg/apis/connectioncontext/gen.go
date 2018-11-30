package connectioncontext

//go:generate protoc -I . -I ../../../../vendor/ connectioncontext.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
