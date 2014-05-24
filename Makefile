CGO_CFLAGS:=-I$(CURDIR)/vendor/libucl/include
CGO_LDFLAGS:=-L$(CURDIR)/vendor/libucl
LIBUCL_NAME=libucl.a
TEST?=./...

# If we're on Windows, we need to change some variables so things compile
# properly.
ifeq ($(OS), Windows_NT)
	LIBUCL_NAME=libucl.dll
endif

export CGO_CFLAGS CGO_LDFLAGS PATH

default: test

libucl: vendor/libucl/$(LIBUCL_NAME)

test: libucl
	go test $(TEST)

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

.PHONY: clean default libucl test
