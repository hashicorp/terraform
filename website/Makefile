######################################################
# NOTE: This file is managed by the Digital Team's   #
# Terraform configuration @ hashicorp/mktg-terraform #
######################################################

.DEFAULT_GOAL := website

# Set the preview mode for the website shell to "developer" or "io"
PREVIEW_MODE ?= developer
REPO ?= terraform

# Enable setting alternate docker tool, e.g. 'make DOCKER_CMD=podman'
DOCKER_CMD ?= docker

CURRENT_GIT_BRANCH=$$(git rev-parse --abbrev-ref HEAD)
LOCAL_CONTENT_DIR=../docs
PWD=$$(pwd)

DOCKER_IMAGE="hashicorp/dev-portal"
DOCKER_IMAGE_LOCAL="dev-portal-local"
DOCKER_RUN_FLAGS=-it \
		--publish "3000:3000" \
		--rm \
		--tty \
		--volume "$(PWD)/docs:/app/docs" \
		--volume "$(PWD)/img:/app/public" \
		--volume "$(PWD)/data:/app/data" \
		--volume "$(PWD)/redirects.js:/app/redirects.js" \
		--volume "next-dir:/app/website-preview/.next" \
		--volume "$(PWD)/.env:/app/.env" \
		--volume "$(PWD)/.env.development:/app/website-preview/.env.development" \
		--volume "$(PWD)/.env.local:/app/website-preview/.env.local" \
		-e "REPO=$(REPO)" \
		-e "PREVIEW_FROM_REPO=$(REPO)" \
		-e "IS_CONTENT_PREVIEW=true" \
		-e "LOCAL_CONTENT_DIR=$(LOCAL_CONTENT_DIR)" \
		-e "CURRENT_GIT_BRANCH=$(CURRENT_GIT_BRANCH)" \
		-e "PREVIEW_MODE=$(PREVIEW_MODE)"

# Default: run this if working on the website locally to run in watch mode.
.PHONY: website
website:
	@echo "==> Downloading latest Docker image..."
	@$(DOCKER_CMD) pull $(DOCKER_IMAGE)
	@echo "==> Starting website..."
	@$(DOCKER_CMD) run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE)

# Use this if you have run `website/build-local` to use the locally built image.
.PHONY: website/local
website/local:
	@echo "==> Starting website from local image..."
	@$(DOCKER_CMD) run $(DOCKER_RUN_FLAGS) $(DOCKER_IMAGE_LOCAL)

# Run this to generate a new local Docker image.
.PHONY: website/build-local
website/build-local:
	@echo "==> Building local Docker image"
	@$(DOCKER_CMD) build https://github.com/hashicorp/dev-portal.git\#main \
		-t $(DOCKER_IMAGE_LOCAL)

