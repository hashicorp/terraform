TEST?=./...
NAME?=$(shell basename "${CURDIR}")
EXTERNAL_TOOLS=\
	github.com/mitchellh/gox

default: test

# test runs the test suite and vets the code.
test: generate
	@echo "==> Running tests..."
	@go list $(TEST) \
		| grep -v "github.com/sethvargo/${NAME}/vendor" \
		| xargs -n1 go test -timeout=60s -parallel=10 ${TESTARGS}

# testrace runs the race checker
testrace: generate
	@echo "==> Running tests (race)..."
	@go list $(TEST) \
		| grep -v "github.com/sethvargo/${NAME}/vendor" \
		| xargs -n1 go test -timeout=60s -race ${TESTARGS}

# updatedeps installs all the dependencies needed to run and build.
updatedeps:
	@sh -c "'${CURDIR}/scripts/deps.sh' '${NAME}'"

# generate runs `go generate` to build the dynamically generated source files.
generate:
	@echo "==> Generating..."
	@find . -type f -name '.DS_Store' -delete
	@go list ./... \
		| grep -v "github.com/hashicorp/${NAME}/vendor" \
		| xargs -n1 go generate

# bootstrap installs the necessary go tools for development/build.
bootstrap:
	@echo "==> Bootstrapping..."
	@for t in ${EXTERNAL_TOOLS}; do \
		echo "--> Installing "$$t"..." ; \
		go get -u "$$t"; \
	done

.PHONY: default test testrace updatedeps generate bootstrap
