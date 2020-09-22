help:
	@echo "test             run test"
	@echo "lint             run lint"

.PHONY: test
test:
	go test -v -cover -coverprofile cover.out
	go tool cover -html=cover.out -o cover.html

.PHONY: lint
lint:
	gofmt -s -w .
	goimports -w .
	golint .
	go vet
