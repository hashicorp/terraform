export GOPATH := $(CURDIR)

COVER_FILE := coverage

all: test

.PHONY: test

test: install-dependencies
	go test -v

cover: install-dependencies install-cover
	go test -v -test.coverprofile="$(COVER_FILE).prof"
	sed -i.bak 's|_'$(GOPATH)'|.|g' $(COVER_FILE).prof
	go tool cover -html=$(COVER_FILE).prof -o $(COVER_FILE).html
	rm $(COVER_FILE).prof*

install-cover:
	go get code.google.com/p/go.tools/cmd/cover

install-dependencies:
	go get github.com/onsi/ginkgo
	go get github.com/onsi/gomega
	go get github.com/streadway/amqp
        # to get Ginkgo CLI
	go install github.com/onsi/ginkgo/ginkgo
