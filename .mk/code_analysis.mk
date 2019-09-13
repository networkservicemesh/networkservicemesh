.PHONY: lint-fix
lint-fix:
	golangci-lint run --new-from-rev=origin/master --fix

.PHONY: lint-install
lint-install:
	GO111MODULE=on go get -u github.com/golangci/golangci-lint/cmd/golangci-lint@v1.18.0

.PHONY: lint-check-diff
lint-check-diff:
	GO111MODULE=on golangci-lint run --new-from-rev=origin/master

.PHONY: lint-check-all
lint-check-all:
	GO111MODULE=on golangci-lint run ./...
