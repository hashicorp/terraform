---
layout: "docker"
page_title: "Docker: docker_volume"
sidebar_current: "docs-docker-resource-volume"
description: |-
  Creates and destroys docker volumes.
---

# docker\_volume

Creates and destroys a volume in Docker. This can be used alongside
[docker\_container](/docs/providers/docker/r/container.html)
to prepare volumes that can be shared across containers.

## Example Usage

```hcl
# Creates a docker volume "shared_volume".
resource "docker_volume" "shared_volume" {
  name = "shared_volume"
}

# Reference the volume with ${docker_volume.shared_volume.name}

```

## Argument Reference

The following arguments are supported:

* `name` - (Optional, string) The name of the Docker volume (generated if not
  provided).
* `driver` - (Optional, string) Driver type for the volume (defaults to local).
* `driver_opts` - (Optional, map of strings) Options specific to the driver.

## Attributes Reference

The following attributes are exported in addition to the above configuration:

* `mountpoint` (string) - The mountpoint of the volume.
