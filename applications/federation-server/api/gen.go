package federation

//go:generate bash -c "protoc -I . federation.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src/ --proto_path=$GOPATH/pkg/mod/"
