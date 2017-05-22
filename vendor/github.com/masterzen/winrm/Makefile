NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
DEPS = $(go list -f '{{range .TestImports}}{{.}} {{end}}' ./... | fgrep -v 'winrm')

all: deps
	@mkdir -p bin/
	@printf "$(OK_COLOR)==> Building$(NO_COLOR)\n"
	@go build github.com/masterzen/winrm

deps:
	@printf "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)\n"
	@go get -d -v ./...
	@echo $(DEPS) | xargs -n1 go get -d

updatedeps:
	go list ./... | xargs go list -f '{{join .Deps "\n"}}' | grep -v github.com/masterzen/winrm | sort -u | xargs go get -f -u -v

clean:
	@rm -rf bin/ pkg/ src/

format:
	go fmt ./...

ci: deps
	@printf "$(OK_COLOR)==> Testing with Coveralls...$(NO_COLOR)\n"
	"$(CURDIR)/scripts/test.sh"

test: deps
	@printf "$(OK_COLOR)==> Testing...$(NO_COLOR)\n"
	go test ./...

.PHONY: all clean deps format test updatedeps
