default: go/networkservice.pb.go

go/networkservice.pb.go: networkservice/networkservice.proto
	protoc -I networkservice networkservice/networkservice.proto --go_out=plugins=grpc:go
