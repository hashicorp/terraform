.PHONY: \
	all \
	lint \
	vet \
	fmtcheck \
	pretest \
	test \
	integration

all: test

lint:
	@ go get -v golang.org/x/lint/golint
	[ -z "$$(golint . | grep -v 'type name will be used as docker.DockerInfo' | grep -v 'context.Context should be the first' | tee /dev/stderr)" ]

vet:
	go vet ./...

fmtcheck:
	[ -z "$$(gofmt -s -d *.go ./testing | tee /dev/stderr)" ]

testdeps:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure -v

pretest: testdeps lint vet fmtcheck

gotest:
	go test -race ./...

test: pretest gotest

integration:
	go test -tags docker_integration -run TestIntegration -v
