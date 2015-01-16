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
	$(eval REF := $(shell sh -c "\
		git symbolic-ref --short HEAD 2>/dev/null \
		|| git rev-parse HEAD"))
	go get -u golang.org/x/tools/cmd/stringer
	go get -f -u -v ./...
	git checkout $(REF)

generate:
	go generate ./...

.PHONY: bin default generate test updatedeps
