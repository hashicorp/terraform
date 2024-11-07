# generate runs `go generate` to build the dynamically generated
# source files, except the protobuf stubs which are built instead with
# "make protobuf".
generate:
	go generate ./...

# We separate the protobuf generation because most development tasks on
# Terraform do not involve changing protobuf files and protoc is not a
# go-gettable dependency and so getting it installed can be inconvenient.
#
# If you are working on changes to protobuf interfaces, run this Makefile
# target to be sure to regenerate all of the protobuf stubs using the expected
# versions of protoc and the protoc Go plugins.
protobuf:
	go run ./tools/protobuf-compile .

fmtcheck:
	"$(CURDIR)/scripts/gofmtcheck.sh"

importscheck:
	"$(CURDIR)/scripts/goimportscheck.sh"

vetcheck:
	@echo "==> Checking that the code complies with go vet requirements"
	@go vet ./...

staticcheck:
	"$(CURDIR)/scripts/staticcheck.sh"

exhaustive:
	"$(CURDIR)/scripts/exhaustive.sh"

copyright:
	"$(CURDIR)/scripts/copyright.sh" --plan

copyrightfix:
	"$(CURDIR)/scripts/copyright.sh"

syncdeps:
	"$(CURDIR)/scripts/syncdeps.sh"

# Run this if working on the website locally to run in watch mode.
website:
	$(MAKE) -C website website

# Use this if you have run `website/build-local` to use the locally built image.
website/local:
	$(MAKE) -C website website/local

# Run this to generate a new local Docker image.
website/build-local:
	$(MAKE) -C website website/build-local

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:

.PHONY: fmtcheck importscheck vetcheck generate protobuf staticcheck syncdeps website website/local website/build-local
