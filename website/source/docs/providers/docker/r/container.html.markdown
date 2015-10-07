---
layout: "docker"
page_title: "Docker: docker_container"
sidebar_current: "docs-docker-resource-container"
description: |-
  Manages the lifecycle of a Docker container.
---

# docker\_container

Manages the lifecycle of a Docker container.

## Example Usage

```
# Start a container
resource "docker_container" "ubuntu" {
  name = "foo"
  image = "${docker_image.ubuntu.latest}"
}

# Find the latest Ubuntu precise image.
resource "docker_image" "ubuntu" {
  name = "ubuntu:precise"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the Docker container.
* `image` - (Required, string) The ID of the image to back this container.
  The easiest way to get this value is to use the `docker_image` resource
  as is shown in the example above.

* `command` - (Optional, list of strings) The command to use to start the
    container.
* `dns` - (Optional, set of strings) Set of DNS servers.
* `env` - (Optional, set of strings) Environmental variables to set.
* `links` - (Optional, set of strings) Set of links for link based
  connectivity between containers that are running on the same host.
* `hostname` - (Optional, string) Hostname of the container.
* `domainname` - (Optional, string) Domain name of the container.
* `must_run` - (Optional, bool) If true, then the Docker container will be
  kept running. If false, then as long as the container exists, Terraform
  assumes it is successful.
* `ports` - (Optional) See [Ports](#ports) below for details.
* `privileged` - (Optional, bool) Run container in privileged mode.
* `publish_all_ports` - (Optional, bool) Publish all ports of the container.
* `volumes` - (Optional) See [Volumes](#volumes) below for details.

<a id="ports"></a>
## Ports

`ports` is a block within the configuration that can be repeated to specify
the port mappings of the container. Each `ports` block supports
the following:

* `internal` - (Required, int) Port within the container.
* `external` - (Required, int) Port exposed out of the container.
* `ip` - (Optional, string) IP address/mask that can access this port.
* `protocol` - (Optional, string) Protocol that can be used over this port,
  defaults to TCP.

<a id="volumes"></a>
## Volumes

`volumes` is a block within the configuration that can be repeated to specify
the volumes attached to a container. Each `volumes` block supports
the following:

* `from_container` - (Optional, string) The container where the volume is
  coming from.
* `container_path` - (Optional, string) The path in the container where the
  volume will be mounted.
* `host_path` - (Optional, string) The path on the host where the volume
  is coming from.
* `read_only` - (Optional, bool) If true, this volume will be readonly.
  Defaults to false.

## Attributes Reference

The following attributes are exported:

 * `ip_address` - The IP address of the container as read from its
   NetworkSettings.
 * `ip_prefix_length` - The IP prefix length of the container as read from its
   NetworkSettings.
 * `gateway` - The network gateway of the container as read from its
   NetworkSettings.
 * `bridge` - The network bridge of the container as read from its
   NetworkSettings.
