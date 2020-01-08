package networkservice

//go:generate bash -c "protoc -I . networkservice.proto --go_out=plugins=grpc:. --proto_path=$GOPATH/src/ --proto_path=$GOPATH/pkg/mod/  --proto_path=$( go list -f '{{ .Dir }}' -m github.com/golang/protobuf )"

//go:generate bash -c "mockgen -destination=../../pkg/tests/mock/networkservice.mg.go -package=mock -self_package=networkservice github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice NetworkServiceServer"
