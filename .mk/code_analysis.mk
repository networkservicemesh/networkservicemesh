.PHONY: lint-fix
lint-fix:
	LOG_LEVEL=error GO111MODULE=on ./scripts/for-each-module.sh "golangci-lint run --new-from-rev=origin/master --fix"

.PHONY: lint-install
lint-install:
	GO111MODULE=on go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.21

.PHONY: lint-check-diff
lint-check-diff:
	LOG_LEVEL=error GO111MODULE=on ./scripts/for-each-module.sh "golangci-lint run ./... --new-from-rev=origin/master"

.PHONY: lint-check-all
lint-check-all:
	LOG_LEVEL=error GO111MODULE=on ./scripts/for-each-module.sh "golangci-lint run ./..."
