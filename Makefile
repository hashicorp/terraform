TEST?=./...
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods -nilfunc -printf -rangeloops -shift -structtags -unsafeptr

default: test

# bin generates the releaseable binaries for Terraform
bin: generate
	@sh -c "'$(CURDIR)/scripts/build.sh'"

# dev creates binaries for testing Terraform locally. These are put
# into ./bin/ as well as $GOPATH/bin
dev: generate
	@TF_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

quickdev: generate
	@TF_QUICKDEV=1 TF_DEV=1 sh -c "'$(CURDIR)/scripts/build.sh'"

# test runs the unit tests and vets the code
test: generate
	TF_ACC= go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4
	@$(MAKE) vet

# testacc runs acceptance tests
testacc: generate
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package"; \
		exit 1; \
	fi
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 45m

# testrace runs the race checker
testrace: generate
	TF_ACC= go test -race $(TEST) $(TESTARGS)

# updatedeps installs all the dependencies that Terraform needs to run
# and build.
updatedeps:
	go get -u github.com/mitchellh/gox
	go get -u golang.org/x/tools/cmd/stringer
	go list ./... \
		| xargs go list -f '{{join .Deps "\n"}}' \
		| grep -v github.com/hashicorp/terraform \
		| sort -u \
		| xargs go get -f -u -v
	cd $GOPATH/src/github.com/MSOpenTech/azure-sdk-for-go && git checkout v1.2
	cd $GOPATH/src/github.com/mitchellh/cli && git checkout e3c2e3d39391e9beb9660ccd6b4bd9a2f38dd8a0
	cd $GOPATH/src/github.com/xanzy/go-cloudstack && git checkout v1.2.0
	cd $GOPATH/src/github.com/pearkes/digitalocean && git checkout d1e3ab42e589ee07827df1d9464dab3aba8b6faa


cover:
	@go tool cover 2>/dev/null; if [ $$? -eq 3 ]; then \
		go get -u golang.org/x/tools/cmd/cover; \
	fi
	go test $(TEST) -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

# vet runs the Go source code static analysis tool `vet` to find
# any common errors.
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

# generate runs `go generate` to build the dynamically generated
# source files.
generate:
	go generate ./...

.PHONY: bin default generate test updatedeps vet
