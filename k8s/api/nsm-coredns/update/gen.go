package update

//go:generate bash -c "protoc -I . update.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src/ --proto_path=$GOPATH/pkg/mod/  --proto_path=$( go list -f '{{ .Dir }}' -m github.com/golang/protobuf )"
