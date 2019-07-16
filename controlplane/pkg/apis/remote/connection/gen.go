package connection

//go:generate bash -c "protoc -I . connection.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src/"
