TEST?=./...

default: test

bin: config/y.go
	@sh -c "'$(CURDIR)/scripts/build.sh'"

dev: config/y.go
	@TF_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

test: config/y.go
	TF_ACC= go test $(TEST) $(TESTARGS) -timeout=10s -parallel=4

testacc: config/y.go
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package"; \
		exit 1; \
	fi
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 45m

testrace: config/y.go
	TF_ACC= go test -race $(TEST) $(TESTARGS)

updatedeps: config/y.go
	# Go 1.4 changed the format of `go get` a bit by requiring the
	# canonical full path. We work around this and just force.
	@if [ $(shell go version | cut -f3 -d" " | cut -f2 -d.) -lt 4 ]; then \
		go get -u -v ./...; \
	else \
		go get -f -u -v ./...; \
	fi

config/y.go: config/expr.y
	cd config/ && \
		go tool yacc -p "expr" expr.y

clean:
	rm config/y.go

.PHONY: bin clean default test updatedeps
