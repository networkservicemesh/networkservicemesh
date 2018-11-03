VERSION	:= $(shell git describe --always --tags --dirty)
COMMIT	:= $(shell git rev-parse HEAD)
DATE	:= $(shell date +'%Y-%m-%dT%H:%M%:z')

CNINFRA_CORE := github.com/ligato/cn-infra/core
LDFLAGS = -ldflags '-X $(CNINFRA_CORE).BuildVersion=$(VERSION) -X $(CNINFRA_CORE).CommitHash=$(COMMIT) -X $(CNINFRA_CORE).BuildDate=$(DATE)'

COVER_DIR ?= /tmp

# Build all
build: examples examples-plugin

# Clean all
clean: clean-examples clean-examples-plugin

# Build examples
examples:
	@echo "=> building examples"
	cd examples/cassandra-lib && go build
	cd examples/etcd-lib && make build
	cd examples/kafka-lib && make build
	cd examples/logs-lib && make build
	cd examples/redis-lib && make build

# Build plugin examples
examples-plugin:
	@echo "=> building plugin examples"
	cd examples/configs-plugin && go build -i -v ${LDFLAGS}
	cd examples/datasync-plugin && go build -i -v ${LDFLAGS}
	cd examples/flags-lib && go build -i -v ${LDFLAGS}
	cd examples/kafka-plugin/hash-partitioner && go build -i -v ${LDFLAGS}
	cd examples/kafka-plugin/manual-partitioner && go build -i -v ${LDFLAGS}
	cd examples/kafka-plugin/post-init-consumer && go build -i -v ${LDFLAGS}
	cd examples/logs-plugin && go build -i -v ${LDFLAGS}
	cd examples/redis-plugin && go build -i -v ${LDFLAGS}
	cd examples/simple-agent && go build -i -v ${LDFLAGS}
	cd examples/statuscheck-plugin && go build -i -v ${LDFLAGS}
	cd examples/prometheus-plugin && go build -i -v ${LDFLAGS}

# Clean examples
clean-examples:
	@echo "=> cleaning examples"
	cd examples/cassandra-lib && rm -f cassandra-lib
	cd examples/etcd-lib && make clean
	cd examples/kafka-lib && make clean
	cd examples/logs-lib && make clean
	cd examples/redis-lib && make clean

# Clean plugin examples
clean-examples-plugin:
	@echo "=> cleaning plugin examples"
	rm -f examples/configs-plugin/configs-plugin
	rm -f examples/datasync-plugin/datasync-plugin
	rm -f examples/flags-lib/flags-lib
	rm -f examples/kafka-plugin/hash-partitioner/hash-partitioner
	rm -f examples/kafka-plugin/manual-partitioner/manual-partitioner
	rm -f examples/kafka-plugin/post-init-consumer/post-init-consumer
	rm -f examples/logs-plugin/logs-plugin
	rm -f examples/redis-plugin/redis-plugin
	rm -f examples/simple-agent/simple-agent
	rm -f examples/statuscheck-plugin/statuscheck-plugin
	rm -f examples/prometheus-plugin/prometheus-plugin

# Get test tools
get-testtools:
	go get github.com/hashicorp/consul

# Run tests
test: get-testtools
	@echo "=> running unit tests"
	go test ./...

# Run script for testing examples
test-examples:
	@echo "=> Testing examples"
	./scripts/test_examples/test_examples.sh
	@echo "=> Testing examples: reactions to disconnect/reconnect of plugins redis, cassandra ..."
	./scripts/test_examples/plugin_reconnect.sh

# Run coverage report
test-cover: get-testtools
	@echo "=> running coverage report"
	go test -covermode=count -coverprofile=${COVER_DIR}/coverage.out ./...
	@echo "=> coverage data generated into ${COVER_DIR}/coverage.out"

test-cover-html: test-cover
	go tool cover -html=${COVER_DIR}/coverage.out -o ${COVER_DIR}/coverage.html
	@echo "=> coverage report generated into ${COVER_DIR}/coverage.html"
	go tool cover -html=${COVER_DIR}/coverage.out

test-cover-xml: test-cover
	gocov convert ${COVER_DIR}/coverage.out | gocov-xml > ${COVER_DIR}/coverage.xml
	@echo "=> coverage report generated into ${COVER_DIR}/coverage.xml"

# Get dependency manager tool
get-dep:
	go get -v github.com/golang/dep/cmd/dep

# Install the project's dependencies
dep-install: get-dep
	dep ensure

# Update the locked versions of all dependencies
dep-update: get-dep
	dep ensure -update

# Get linter tools
get-linters:
	@echo "=> installing linters"
	go get -v github.com/alecthomas/gometalinter
	gometalinter --install

# Run linters
lint: get-linters
	@echo "=> running code analysis"
	./scripts/static_analysis.sh golint vet

# Format code
format:
	@echo "=> formatting the code"
	./scripts/gofmt.sh

# Get link check tool
get-linkcheck:
	sudo apt-get install npm
	npm install -g markdown-link-check

# Validate links in markdown files
check-links: get-linkcheck
	@echo "=> checking links"
	./scripts/check_links.sh

.PHONY: build clean \
	examples examples-plugin clean-examples clean-examples-plugin test test-examples \
	test-cover test-cover-html test-cover-xml \
	get-dep dep-install dep-update \
	get-linters lint format \
	get-linkcheck check-links
