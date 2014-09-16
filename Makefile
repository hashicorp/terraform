TEST?=./...

default: test

bin: config/y.go
	@sh -c "$(CURDIR)/scripts/build.sh"

dev: config/y.go
	@TF_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"

test: config/y.go
	TF_ACC= go test $(TEST) $(TESTARGS) -timeout=10s -parallel=4

testacc: config/y.go
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package"; \
		exit 1; \
	fi
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 30m

testrace: config/y.go
	TF_ACC= go test -race $(TEST) $(TESTARGS)

updatedeps: config/y.go
	go get -u -v ./...

config/y.go: config/expr.y
	cd config/ && \
		go tool yacc -p "expr" expr.y

clean:
	rm config/y.go

.PHONY: bin clean default test updatedeps
