---
layout: "docker"
page_title: "Docker: docker_registry_image"
sidebar_current: "docs-docker-datasource-registry-image"
description: |-
  Finds the latest available sha256 digest for a docker image/tag from a registry.
---

# docker\_registry\_image

-> **Note:** The initial (current) version of this data source can reliably read only **public** images **from the official Docker Hub Registry**.

Reads the image metadata from a Docker Registry. Used in conjunction with the
[docker\_image](/docs/providers/docker/r/image.html) resource to keep an image up
to date on the latest available version of the tag.

## Example Usage

```hcl
data "docker_registry_image" "ubuntu" {
  name = "ubuntu:precise"
}

resource "docker_image" "ubuntu" {
  name          = "${data.docker_registry_image.ubuntu.name}"
  pull_triggers = ["${data.docker_registry_image.ubuntu.sha256_digest}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker image, including any tags. e.g. `alpine:latest`

## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `sha256_digest` (string) - The content digest of the image, as stored on the registry.
