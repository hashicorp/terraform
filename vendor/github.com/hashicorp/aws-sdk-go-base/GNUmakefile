default: test lint

lint:
	@echo "==> Checking source code against linters..."
	@golangci-lint run ./...

test:
	go test -timeout=30s -parallel=4 ./...

tools:
	GO111MODULE=off go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: lint test tools
