help:
	@echo "test             run test"
	@echo "lint             run lint"
	@echo "example          run examples"

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

.PHONY: example
example:
	cd example && sh test.sh
ci-test:
	cd costesting && go test -v 
