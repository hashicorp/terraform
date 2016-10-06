---
layout: "docker"
page_title: "Provider: Docker"
sidebar_current: "docs-docker-index"
description: |-
  The Docker provider is used to interact with Docker containers and images.
---

# Docker Provider

The Docker provider is used to interact with Docker containers and images.
It uses the Docker API to manage the lifecycle of Docker containers. Because
the Docker provider uses the Docker API, it is immediately compatible not
only with single server Docker but Swarm and any additional Docker-compatible
API hosts.

Use the navigation to the left to read about the available resources.

<div class="alert alert-block alert-info">
<strong>Note:</strong> The Docker provider is new as of Terraform 0.4.
It is ready to be used but many features are still being added. If there
is a Docker feature missing, please report it in the GitHub repo.
</div>

## Example Usage

```
# Configure the Docker provider
provider "docker" {
    host = "tcp://127.0.0.1:2376/"
}

# Create a container
resource "docker_container" "foo" {
    image = "${docker_image.ubuntu.latest}"
    name = "foo"
}

resource "docker_image" "ubuntu" {
    name = "ubuntu:latest"
}
```

## Registry Credentials

The initial (current) version of the Docker provider **doesn't** support registry authentication.
This limits any use cases to public images for now.

## Argument Reference

The following arguments are supported:

* `host` - (Required) This is the address to the Docker host. If this is
  blank, the `DOCKER_HOST` environment variable will also be read.

* `cert_path` - (Optional) Path to a directory with certificate information
  for connecting to the Docker host via TLS. If this is blank, the
  `DOCKER_CERT_PATH` will also be checked.

~> **NOTE on Certificates and `docker-machine`:**  As per [Docker Remote API
documentation](https://docs.docker.com/engine/reference/api/docker_remote_api/),
in any docker-machine environment, the Docker daemon uses an encrypted TCP
socket (TLS) and requires `cert_path` for a successful connection. As an alternative,
if using `docker-machine`, run `eval $(docker-machine env <machine-name>)` prior
to running Terraform, and the host and certificate path will be extracted from
the environment.
