---
layout: "docker"
page_title: "Docker: docker_image"
sidebar_current: "docs-docker-resource-image"
description: |-
  Pulls a Docker image to a given Docker host.
---

# docker\_image

-> **Note:** The initial (current) version of this resource can only pull **public** images **from the official Docker Hub Registry**.

Pulls a Docker image to a given Docker host from a Docker Registry.

This resource will *not* pull new layers of the image automatically unless used in
conjunction with [`docker_registry_image`](/docs/providers/docker/d/registry_image.html)
data source to update the `pull_triggers` field.

## Example Usage

```hcl
# Find the latest Ubuntu precise image.
resource "docker_image" "ubuntu" {
  name = "ubuntu:precise"
}

# Access it somewhere else with ${docker_image.ubuntu.latest}

```

### Dynamic image

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

* `name` - (Required, string) The name of the Docker image, including any tags.
* `keep_locally` - (Optional, boolean) If true, then the Docker image won't be
  deleted on destroy operation. If this is false, it will delete the image from
  the docker local storage on destroy operation.
* `pull_triggers` - (Optional, list of strings) List of values which cause an
  image pull when changed. This is used to store the image digest from the
  registry when using the `docker_registry_image` [data source](/docs/providers/docker/d/registry_image.html)
  to trigger an image update.
* `pull_trigger` - **Deprecated**, use `pull_triggers` instead.


## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `latest` (string) - The ID of the image.
