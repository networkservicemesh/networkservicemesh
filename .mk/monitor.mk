.PHONY: mon
mon: mon-server mon-client

.PHONY: mon-server
mon-server:
	${GO_BUILD} -o $(GOPATH)/bin/mon-server ./test/applications/skydive/monitor-server.go

.PHONY: mon-client
mon-client:
	${GO_BUILD} -o $(GOPATH)/bin/mon-client ./test/applications/skydive/monitor-client.go
