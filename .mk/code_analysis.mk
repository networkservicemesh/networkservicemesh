.PHONY: lint-fix
lint-fix:
	golangci-lint run --new-from-rev=origin/master --fix

.PHONY: lint-install
lint-install:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.16.0

.PHONY: lint-check-diff
lint-check-diff:
	golangci-lint run --new-from-rev=origin/master

.PHONY: lint-check-all
lint-check-all:
	golangci-lint run ./...
