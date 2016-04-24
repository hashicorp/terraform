TEST?=.
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

default: test

# get dependencies
updatedeps:
	go list ./... \
        | xargs go list -f '{{join .Deps "\n"}}' \
		| grep -v go-datadog-api\
        | grep -v '/internal/' \
        | sort -u \
        | xargs go get -f -u -v

# test runs the unit tests and vets the code
test:
	go test . $(TESTARGS) -v -timeout=30s -parallel=4
	@$(MAKE) vet

# testacc runs acceptance tests
testacc:
	go test integration/* -v $(TESTARGS) -timeout 90m

# testrace runs the race checker
testrace:
	go test -race $(TEST) $(TESTARGS)

# vet runs the Go source code static analysis tool `vet` to find
# any common errors.
vet:
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@echo "go tool vet $(VETARGS) $(TEST) "
	@go tool vet $(VETARGS) $(TEST) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

.PHONY: default test testacc updatedeps vet
