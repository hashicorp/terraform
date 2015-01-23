TEST?=./...
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

default: test

bin: generate
	@sh -c "'$(CURDIR)/scripts/build.sh'"

dev: generate
	@TF_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

test: generate
	TF_ACC= go test $(TEST) $(TESTARGS) -timeout=10s -parallel=4
	@$(MAKE) vet

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
	go get -u github.com/mitchellh/gox
	go get -u golang.org/x/tools/cmd/stringer
	go get -u golang.org/x/tools/cmd/vet
	go get -f -u -v ./...
	git checkout $(REF)

vet:
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@echo "go tool vet $(VETARGS) ."
	@go tool vet $(VETARGS) . ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for reviewal."; \
	fi

generate:
	go generate ./...

.PHONY: bin default generate test updatedeps vet
