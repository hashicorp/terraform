TEST?=./...

default: test

bin: generate
	@sh -c "'$(CURDIR)/scripts/build.sh'"

dev: generate
	@TF_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

test: generate
	TF_ACC= go test $(TEST) $(TESTARGS) -timeout=10s -parallel=4

testacc: generate
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package"; \
		exit 1; \
	fi
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 45m

testrace: generate
	TF_ACC= go test -race $(TEST) $(TESTARGS)

updatedeps:
	go get -u golang.org/x/tools/cmd/stringer
	# Go 1.4 changed the format of `go get` a bit by requiring the
	# canonical full path. We work around this and just force.
	@if [ $(shell go version | cut -f3 -d" " | cut -f2 -d.) -lt 4 ]; then \
		go get -u -v ./...; \
	else \
		go get -f -u -v ./...; \
	fi

generate:
	go generate ./...

.PHONY: bin default generate test updatedeps
