PACKAGES = $(shell go list ./... | grep -v '/vendor/')
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods \
         -nilfunc -printf -rangeloops -shift -structtags -unsafeptr
EXTERNAL_TOOLS=\
	github.com/kardianos/govendor \
	github.com/mitchellh/gox \
	golang.org/x/tools/cmd/cover \
	github.com/axw/gocov/gocov \
	gopkg.in/matm/v1/gocov-html \
	github.com/ugorji/go/codec/codecgen

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

all: test

dev: format generate
	@NOMAD_DEV=1 sh -c "'$(PWD)/scripts/build.sh'"

bin: generate
	@sh -c "'$(PWD)/scripts/build.sh'"

release:
	@$(MAKE) bin

cov:
	gocov test ./... | gocov-html > /tmp/coverage.html
	open /tmp/coverage.html

test: generate
	@echo "--> Running go fmt" ;
	@if [ -n "`go fmt ${PACKAGES}`" ]; then \
		echo "[ERR] go fmt updated formatting. Please commit formatted code first."; \
		exit 1; \
	fi
	@sh -c "'$(PWD)/scripts/test.sh'"
	@$(MAKE) vet

cover:
	go list ./... | xargs -n1 go test --cover

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

generate:
	@echo "--> Running go generate"
	@go generate $(PACKAGES)
	@sed -e 's|github.com/hashicorp/nomad/vendor/github.com/ugorji/go/codec|github.com/ugorji/go/codec|' nomad/structs/structs.generated.go >> structs.gen.tmp
	@mv structs.gen.tmp nomad/structs/structs.generated.go

vet:
	@go tool vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@echo "--> Running go tool vet $(VETARGS) ${GOFILES_NOVENDOR}"
	@go tool vet $(VETARGS) ${GOFILES_NOVENDOR} ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "[LINT] Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
	fi

	@git grep -n `echo "log"".Print"` | grep -v 'vendor/' ; if [ $$? -eq 0 ]; then \
		echo "[LINT] Found "log"".Printf" calls. These should use Nomad's logger instead."; \
	fi

web:
	./scripts/website_run.sh

web-push:
	./scripts/website_push.sh

# bootstrap the build by downloading additional tools
bootstrap:
	@for tool in  $(EXTERNAL_TOOLS) ; do \
		echo "Installing $$tool" ; \
    go get $$tool; \
	done

install: bin/nomad
	install -o root -g wheel -m 0755 ./bin/nomad /usr/local/bin/nomad

travis:
	@sh -c "'$(PWD)/scripts/travis.sh'"

.PHONY: all bin cov integ test vet web web-push test-nodep
