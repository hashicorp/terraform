TEST?=./...

default: test

fmt: hcl/y.go json/y.go
	go fmt ./...

test: hcl/y.go json/y.go
	go test $(TEST) $(TESTARGS)

hcl/y.go: hcl/parse.y
	cd hcl && \
		go tool yacc -p "hcl" parse.y

json/y.go: json/parse.y
	cd json/ && \
		go tool yacc -p "json" parse.y

clean:
	rm -f hcl/y.go
	rm -f json/y.go

.PHONY: default test
