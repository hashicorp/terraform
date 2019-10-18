VERSION?="0.3.32"
TEST?=./...
GOFMT_FILES?=$$(find . -not -path "./vendor/*" -type f -name '*.go')
WEBSITE_REPO=github.com/hashicorp/terraform-website

default: test

# bin generates the releaseable binaries for Terraform
bin: fmtcheck generate
	@TF_RELEASE=1 sh -c "'$(CURDIR)/scripts/build.sh'"

# dev creates binaries for testing Terraform locally. These are put
# into ./bin/ as well as $GOPATH/bin
dev: fmtcheck generate
	go install -mod=vendor .

quickdev: generate
	go install -mod=vendor .

# Shorthand for building and installing just one plugin for local testing.
# Run as (for example): make plugin-dev PLUGIN=provider-aws
plugin-dev: generate
	go install github.com/hashicorp/terraform/builtin/bins/$(PLUGIN)
	mv $(GOPATH)/bin/$(PLUGIN) $(GOPATH)/bin/terraform-$(PLUGIN)

# test runs the unit tests
# we run this one package at a time here because running the entire suite in
# one command creates memory usage issues when running in Travis-CI.
test: fmtcheck generate
	go list -mod=vendor $(TEST) | xargs -t -n4 go test $(TESTARGS) -mod=vendor -timeout=2m -parallel=4

# testacc runs acceptance tests
testacc: fmtcheck generate
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make testacc TEST=./builtin/providers/test"; \
		exit 1; \
	fi
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -mod=vendor -timeout 120m

# e2etest runs the end-to-end tests against a generated Terraform binary
# The TF_ACC here allows network access, but does not require any special
# credentials since the e2etests use local-only providers such as "null".
e2etest: generate
	TF_ACC=1 go test -mod=vendor -v ./command/e2etest

test-compile: fmtcheck generate
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./builtin/providers/test"; \
		exit 1; \
	fi
	go test -mod=vendor -c $(TEST) $(TESTARGS)

# testrace runs the race checker
testrace: fmtcheck generate
	TF_ACC= go test -mod=vendor -race $(TEST) $(TESTARGS)

cover:
	go test $(TEST) -coverprofile=coverage.out
	go tool cover -html=coverage.out
	rm coverage.out

# generate runs `go generate` to build the dynamically generated
# source files, except the protobuf stubs which are built instead with
# "make protobuf".
generate:
	GOFLAGS=-mod=vendor go generate ./...
	# go fmt doesn't support -mod=vendor but it still wants to populate the
	# module cache with everything in go.mod even though formatting requires
	# no dependencies, and so we're disabling modules mode for this right
	# now until the "go fmt" behavior is rationalized to either support the
	# -mod= argument or _not_ try to install things.
	GO111MODULE=off go fmt command/internal_plugin_list.go > /dev/null

# We separate the protobuf generation because most development tasks on
# Terraform do not involve changing protobuf files and protoc is not a
# go-gettable dependency and so getting it installed can be inconvenient.
#
# If you are working on changes to protobuf interfaces you may either use
# this target or run the individual scripts below directly.
protobuf:
	bash scripts/protobuf-check.sh
	bash internal/tfplugin5/generate.sh
	bash plans/internal/planproto/generate.sh

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

website:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), get-ting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	$(eval WEBSITE_PATH := $(GOPATH)/src/$(WEBSITE_REPO))
	@echo "==> Starting core website in Docker..."
	@docker run \
		--interactive \
		--rm \
		--tty \
		--publish "4567:4567" \
		--publish "35729:35729" \
		--volume "$(shell pwd)/website:/website" \
		--volume "$(shell pwd):/ext/terraform" \
		--volume "$(WEBSITE_PATH)/content:/terraform-website" \
		--volume "$(WEBSITE_PATH)/content/source/assets:/website/docs/assets" \
		--volume "$(WEBSITE_PATH)/content/source/layouts:/website/docs/layouts" \
		--workdir /terraform-website \
		hashicorp/middleman-hashicorp:${VERSION}

website-test:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), get-ting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	$(eval WEBSITE_PATH := $(GOPATH)/src/$(WEBSITE_REPO))
	@echo "==> Testing core website in Docker..."
	-@docker stop "tf-website-core-temp"
	@docker run \
		--detach \
		--rm \
		--name "tf-website-core-temp" \
		--publish "4567:4567" \
		--volume "$(shell pwd)/website:/website" \
		--volume "$(shell pwd):/ext/terraform" \
		--volume "$(WEBSITE_PATH)/content:/terraform-website" \
		--volume "$(WEBSITE_PATH)/content/source/assets:/website/docs/assets" \
		--volume "$(WEBSITE_PATH)/content/source/layouts:/website/docs/layouts" \
		--workdir /terraform-website \
		hashicorp/middleman-hashicorp:${VERSION}
	$(WEBSITE_PATH)/content/scripts/check-links.sh "http://127.0.0.1:4567" "/" "/docs/providers/*"
	@docker stop "tf-website-core-temp"

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:

.PHONY: bin cover default dev e2etest fmt fmtcheck generate protobuf plugin-dev quickdev test-compile test testacc testrace vendor-status website website-test
