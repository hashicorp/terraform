CGO_CFLAGS:=-I$(CURDIR)/vendor/libucl/include
CGO_LDFLAGS:=-L$(CURDIR)/vendor/libucl
LIBUCL_NAME=libucl.a
TEST?=./...

# Windows-only
ifeq ($(OS), Windows_NT)
	# The Libucl library is named libucl.dll
	LIBUCL_NAME=libucl.dll

	# Add the current directory on the path so the DLL is available.
	export PATH := $(CURDIR):$(PATH)
endif

export CGO_CFLAGS CGO_LDFLAGS PATH

default: test

bin: config/y.go libucl
	@sh -c "$(CURDIR)/scripts/build.sh"

dev: config/y.go libucl
	@TF_DEV=1 sh -c "$(CURDIR)/scripts/build.sh"

libucl: vendor/libucl/$(LIBUCL_NAME)

test: config/y.go libucl
	TF_ACC= go test $(TEST) $(TESTARGS) -timeout=10s

testacc: config/y.go libucl
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package"; \
		exit 1; \
	fi
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 30m

testrace: config/y.go libucl
	TF_ACC= go test -race $(TEST) $(TESTARGS)

updatedeps: config/y.go libucl
	go get -u -v ./...

config/y.go: config/expr.y
	cd config/ && \
		go tool yacc -p "expr" expr.y

vendor/libucl/libucl.a: vendor/libucl
	cd vendor/libucl && \
		cmake cmake/ && \
		make

vendor/libucl/libucl.dll: vendor/libucl
	cd vendor/libucl && \
		$(MAKE) -f Makefile.w32 && \
		cp .obj/libucl.dll . && \
		cp libucl.dll $(CURDIR)

vendor/libucl:
	rm -rf vendor/libucl
	mkdir -p vendor/libucl
	git clone https://github.com/hashicorp/libucl.git vendor/libucl
	cd vendor/libucl && \
		git checkout fix-win32-compile

clean:
	rm config/y.go
	rm -rf vendor

.PHONY: clean default libucl test updatedeps
