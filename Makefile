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

# ci mirrors the "Quick Checks" workflow in .github/workflows/checks.yml.
# ci-consistency runs first to ensure generated files are up-to-date before
# running tests. Test suites then run in parallel via a sub-make.
ci:
	$(MAKE) ci-consistency
	$(MAKE) -j ci-unit-tests ci-race-tests ci-e2e-tests

ci-unit-tests:
	@for dir in $$(go list -m -f '{{.Dir}}' github.com/hashicorp/terraform/...); do \
		(cd "$$dir" && go test -cover "./..."); \
	done

ci-race-tests:
	@go test -race ./internal/terraform ./internal/command ./internal/states

ci-e2e-tests:
	@TF_ACC=1 go test -v ./internal/command/e2etest

# ci-consistency runs in three phases to allow parallelism within each phase
# while maintaining correct ordering:
#   Phase 1: syncdeps (go mod tidy across all modules)
#   Phase 2: generate + protobuf (code generation, can run in parallel)
#   Phase 3: all checks (read-only, can run in parallel)
ci-consistency:
	$(MAKE) syncdeps
	$(MAKE) -j generate protobuf
	$(MAKE) -j fmtcheck importscheck vetcheck copyright staticcheck exhaustive
	@echo "==> Checking for inconsistencies (Quick Checks workflow)"
	@changed="$$(git status --porcelain)"; \
	if [ -n "$$changed" ]; then \
		git diff; \
		echo >&2 "ERROR: Generated files are inconsistent. Run 'make syncdeps', 'make generate', and 'make protobuf' locally and then commit the updated files."; \
		printf >&2 'Affected files:\n%s\n' "$$changed"; \
		exit 1; \
	fi


.PHONY: ci ci-unit-tests ci-race-tests ci-e2e-tests ci-consistency fmtcheck importscheck vetcheck generate protobuf staticcheck exhaustive copyright copyrightfix syncdeps
