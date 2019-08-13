.PHONY: format
format:
	GO111MODULE=on goimports -w -local github.com/networkservicemesh/networkservicemesh -d `find . -type f -name '*.go' -not -name '*.pb.go' -not -path "./vendor/*"`

.PHONY: install-formatter
install-formatter:
	GO111MODULE=off go get -u golang.org/x/tools/cmd/goimports
