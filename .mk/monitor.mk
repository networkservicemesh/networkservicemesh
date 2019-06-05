.PHONY: mon
mon: mon-server mon-client

.PHONY: mon-server
mon-server:
	CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o $(GOPATH)/bin/mon-server ./test/applications/skydive/monitor-server.go

.PHONY: mon-client
mon-client:
	CGO_ENABLED=0 GOOS=linux go build -ldflags '-extldflags "-static"' -o $(GOPATH)/bin/mon-client ./test/applications/skydive/monitor-client.go
