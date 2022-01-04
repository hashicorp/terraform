WEBSITE_REPO=github.com/hashicorp/terraform-website
VERSION?="0.3.44"

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
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

staticcheck:
	@sh -c "'$(CURDIR)/scripts/staticcheck.sh'"

exhaustive:
	@sh -c "'$(CURDIR)/scripts/exhaustive.sh'"

website:
	@echo "==> Downloading latest Docker image..."
	@docker pull hashicorp/terraform-website:full
	@echo "==> Starting core website in Docker..."
	@docker run \
		--interactive \
		--rm \
		--tty \
		--workdir "/website" \
		--volume "$(shell pwd):/website/ext/terraform" \
		--publish "3000:3000" \
		hashicorp/terraform-website:full \
		npm start

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:

.PHONY: fmtcheck generate protobuf website website-test staticcheck
