.PHONY: \
	all \
	vendor \
	lint \
	vet \
	fmt \
	fmtcheck \
	pretest \
	test \
	integration \
	cov \
	clean

PKGS = . ./testing

all: test

vendor:
	@ go get -v github.com/mjibson/party
	party -d external -c -u

lint:
	@ go get -v github.com/golang/lint/golint
	@for file in $$(git ls-files '*.go' | grep -v 'external/'); do \
		export output="$$(golint $${file} | grep -v 'type name will be used as docker.DockerInfo')"; \
		[ -n "$${output}" ] && echo "$${output}" && export status=1; \
	done; \
	exit $${status:-0}

vet:
	go vet $(PKGS)

fmt:
	gofmt -s -w $(PKGS)

fmtcheck:
	@ export output=$$(gofmt -s -d $(PKGS)); \
		[ -n "$${output}" ] && echo "$${output}" && export status=1; \
		exit $${status:-0}

pretest: lint vet fmtcheck

gotest:
	go test $(GO_TEST_FLAGS) $(PKGS)

test: pretest gotest

integration:
	go test -tags docker_integration -run TestIntegration -v

cov:
	@ go get -v github.com/axw/gocov/gocov
	@ go get golang.org/x/tools/cmd/cover
	gocov test | gocov report

clean:
	go clean $(PKGS)
