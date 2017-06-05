VERSION?="0.3.22"

build:
ifeq ($(MM_ENV),local_build)
	@echo "==> modifying source/ for local build"
	@./scripts/prepare-local-build.sh
endif
	@echo "==> Starting build in Docker..."
	@docker run \
		--interactive \
		--rm \
		--tty \
		--volume "$(shell pwd):/website" \
		-e MM_ENV=$(MM_ENV) \
		hashicorp/middleman-hashicorp:${VERSION} \
		bundle exec middleman build --verbose --clean
ifeq ($(MM_ENV),local_build)
	@echo "==> removing changes to source/"
	@git checkout source
endif


website:
	@echo "==> Starting website in Docker..."
	@docker run \
		--interactive \
		--rm \
		--tty \
		--publish "4567:4567" \
		--publish "35729:35729" \
		--volume "$(shell pwd):/website" \
		hashicorp/middleman-hashicorp:${VERSION}

.PHONY: build website
