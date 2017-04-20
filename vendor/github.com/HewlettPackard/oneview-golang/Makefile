# # Plain make targets if not requested inside a container

USE_CONTAINER ?= true

define noop_targets
	@make -pn | sed -rn '/^[^# \t\.%].*:[^=]?/p'|grep -v '='| grep -v '(%)'| grep -v '/'| awk -F':' '{print $$1}'|sort -u;
endef

define setup_testcase_creds
  if [ ! -z "$(TEST_CASES)" ]; then \
    test_cred=$$(echo '$(TEST_CASES)'| awk -F':' '{print $$2}'); \
    test_dir=$$(dirname $$test_cred); \
    echo "setting up testcases cred for container test acceptance"; \
    test -z "$(shell docker ps -a --format '{{.Names}}' | grep '$(DOCKER_IMAGE_NAME)$$')" || \
      docker rm -f $(DOCKER_IMAGE_NAME);  \
    test -z "$(shell docker ps -a --format '{{.Names}}' | grep 'oneview-golang-testaccept$$')" || \
      docker rm -f oneview-golang-testaccept;  \
    test -z "$(shell docker volume ls | awk '{print $2}' | grep 'oneview-golang-testcreds$$')" || \
      docker volume rm oneview-golang-testcreds; \
    docker create -v oneview-golang-testcreds:$$test_dir --name oneview-golang-testaccept alpine sh; \
    docker cp $$test_cred oneview-golang-testaccept:$$test_cred; \
  else \
    echo "skipping no test_cases to setup creds for"; \
  fi
endef

GET_TESTCASE_VOLUME ?= $(shell [ ! -z "$(TEST_CASES)" ] && echo "--volumes-from oneview-golang-testaccept")

include Makefile.inc

ifneq (,$(findstring test-integration,$(MAKECMDGOALS)))
	include mk/main.mk
else ifeq ($(USE_CONTAINER),false)
	include mk/main.mk
else
# Otherwise, with docker, swallow all targets and forward into a container
DOCKER_IMAGE_NAME := "oneview-golang-build"
DOCKER_CONTAINER_NAME := oneview-golang-build-container
# get the dockerfile from docker/machine project so we stay in sync with the versions they use for go
DOCKER_FILE_URL := file://$(PREFIX)/Dockerfile
DOCKER_FILE := .dockerfile.oneview

noop:
	@echo When using 'USE_CONTAINER' use a "make <target>"
	@echo
	@echo Possible targets
	@echo
	$(call noop_targets)

clean: gen-dockerfile

# use this to run a single test case
# ie;
# TEST_CASES=EGSL_HOUSTB200_LAB:/home/docker/git/\
# github.com/HewlettPackard/docker-machine-oneview\
# /tools/oneview-machine/creds.env \
# ONEVIEW_DEBUG=true make test-case TEST_RUN='-test.run=TestGetAPIVersion'
test-case: gen-dockerfile
	@$(call setup_testcase_creds)
	make test-acceptance TEST_RUN='-test.run=TestGetAPIVersion'

test: gen-dockerfile

%:
		export GO15VENDOREXPERIMENT=1
		docker build -f $(DOCKER_FILE) -t $(DOCKER_IMAGE_NAME) .

		test -z '$(shell docker ps -a | grep $(DOCKER_CONTAINER_NAME))' || docker rm -f $(DOCKER_CONTAINER_NAME)

		docker run --name $(DOCKER_CONTAINER_NAME) \
		    $(GET_TESTCASE_VOLUME) \
		    -e DEBUG \
		    -e STATIC \
		    -e VERBOSE \
		    -e BUILDTAGS \
		    -e PARALLEL \
		    -e COVERAGE_DIR \
		    -e TARGET_OS \
		    -e TARGET_ARCH \
		    -e PREFIX \
		    -e GO15VENDOREXPERIMENT \
				-e TEST_CASES \
		    -e TEST_RUN \
		    -e ONEVIEW_DEBUG \
		    -e GH_USER \
		    -e GH_REPO \
		    -e USE_CONTAINER=false \
		    $(DOCKER_IMAGE_NAME) \
		    make $@

endif

include mk/utils/dockerfile.mk
include mk/utils/glide.mk
