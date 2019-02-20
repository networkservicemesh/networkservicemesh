package connectioncontext

//go:generate protoc -I . connectioncontext.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src
