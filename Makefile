CGO_CFLAGS:=-I$(CURDIR)/vendor/libucl/include
CGO_LDFLAGS:=-L$(CURDIR)/vendor/libucl
LIBUCL_NAME=libucl.a
TEST?=./...
TESTARGS?=-timeout=5s

# Windows-only
ifeq ($(OS), Windows_NT)
	# The Libucl library is named libucl.dll
	LIBUCL_NAME=libucl.dll

	# Add the current directory on the path so the DLL is available.
	export PATH := $(CURDIR):$(PATH)
endif

export CGO_CFLAGS CGO_LDFLAGS PATH

default: test

dev: libucl
	@sh -c "$(CURDIR)/scripts/build.sh"

libucl: vendor/libucl/$(LIBUCL_NAME)

test: libucl
	go test $(TEST) $(TESTARGS)

testrace: libucl
	go test -race $(TEST) $(TESTARGS)

updatedeps:
	go get -u -v ./...

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
	git clone https://github.com/vstakhov/libucl.git vendor/libucl

clean:
	rm -rf vendor

.PHONY: clean default libucl test updatedeps
