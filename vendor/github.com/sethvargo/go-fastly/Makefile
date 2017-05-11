# Metadata about this makefile and position
MKFILE_PATH := $(lastword $(MAKEFILE_LIST))
CURRENT_DIR := $(dir $(realpath $(MKFILE_PATH)))
CURRENT_DIR := $(CURRENT_DIR:/=)

# Get the project metadata
GOVERSION := 1.8
PROJECT := github.com/sethvargo/go-fastly
OWNER := $(dir $(PROJECT))
OWNER := $(notdir $(OWNER:/=))
NAME := $(notdir $(PROJECT))
EXTERNAL_TOOLS =

# List of tests to run
TEST ?= ./...

# List all our actual files, excluding vendor
GOFILES = $(shell go list $(TEST) | grep -v /vendor/)

# Tags specific for building
GOTAGS ?=

# Number of procs to use
GOMAXPROCS ?= 4

# bootstrap installs the necessary go tools for development or build
bootstrap:
	@echo "==> Bootstrapping ${PROJECT}..."
	@for t in ${EXTERNAL_TOOLS}; do \
		echo "--> Installing $$t" ; \
		go get -u "$$t"; \
	done

# deps gets all the dependencies for this repository and vendors them.
deps:
	@echo "==> Updating dependencies..."
	@docker run \
		--interactive \
		--tty \
		--rm \
		--dns=8.8.8.8 \
		--env="GOMAXPROCS=${GOMAXPROCS}" \
		--workdir="/go/src/${PROJECT}" \
		--volume="${CURRENT_DIR}:/go/src/${PROJECT}" \
		"golang:${GOVERSION}" /usr/bin/env sh -c "scripts/deps.sh"

# generate runs the code generator
generate:
	@echo "==> Generating ${PROJECT}..."
	@go generate ${GOFILES}

# test runs the test suite
test:
	@echo "==> Testing ${PROJECT}..."
	@go test -timeout=60s -parallel=20 -tags="${GOTAGS}" ${GOFILES} ${TESTARGS}

# test-race runs the race checker
test-race:
	@echo "==> Testing ${PROJECT} (race)..."
	@go test -timeout=60s -race -tags="${GOTAGS}" ${GOFILES} ${TESTARGS}

.PHONY: bootstrap deps generate test test-race
