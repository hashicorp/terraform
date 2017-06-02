SOURCEDIR=.
SOURCES = $(shell find $(SOURCEDIR) -name '*.go')
VERSION=$(git describe --always --tags)
BINARY=bin/runscope

bin: $(BINARY)

$(BINARY): $(SOURCES)
	go build -o $(BINARY) command/*

.PHONY: build
build:
	go get github.com/golang/lint/golint
	go test $(go list ./... | grep -v /vendor/)
	go vet $(go list ./... | grep -v /vendor/)
	golint $(go list ./... | grep -v /vendor/)

.PHONY: test
test:
	go test ./...