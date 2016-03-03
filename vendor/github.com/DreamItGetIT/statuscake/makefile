.PHONY: default lint test

default: lint test

lint:
	@golint ./...
	@go vet ./...

test:
	go test ${GOTEST_ARGS} ./...

