TEST?=./...

default: test

# test runs the test suite and vets the code
test: generate
	go list $(TEST) | xargs -n1 go test -timeout=30s -parallel=12 $(TESTARGS)

# updatedeps installs all the dependencies the library needs to run and build
updatedeps:
	go list ./... \
		| xargs go list -f '{{ join .Deps "\n" }}{{ printf "\n" }}{{ join .TestImports "\n" }}' \
		| grep -v github.com/sethvargo/go-fastly \
		| xargs go get -f -u -v

# generate runs `go generate` to build the dynamically generated source files
generate:
	find . -type f -name '.DS_Store' -delete
	go generate ./...

.PHONY: default bin dev dist test testrace updatedeps generate
