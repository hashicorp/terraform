---
layout: "dockercloud"
page_title: "Docker Cloud: dockercloud_service"
sidebar_current: "docs-dockercloud-resource-service"
description: |-
  Provides a Docker Cloud service resource.
---

# dockercloud\_service

Provides a Docker Cloud service resource.

## Example Usage

```
# Create a new node cluster
resource "dockercloud_node_cluster" "default" {
    name = "dev"
    node_provider = "aws"
    region = "us-east-1"
    size = "t2.micro"
}

# Create a sample web service
resource "dockercloud_service" "web" {
    name = "web_server"
    image = "python:3.2"
    entrypoint = "python -m http.server"

    # Explicitly set dependency on the node cluster
    depends_on = ["dockercloud_node_cluster.default"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the service (cannot contain underscores).
* `image` - (Required) The image to start the container with.
* `entrypoint` - (Optional) The entrypoint used at container startup.
* `command` - (Optional) The command used at container startup.
* `workdir` - (Optional) The directory to start from at container startup.
* `pid` - (Optional) The process ID to run the command as.
* `net` - (Optional) The networking type of the container.
* `privileged` - (Optional, bool) Run container in privileged mode.
* `env` - (Optional, set of strings) Environmental variables to set (in "key=value" format).
* `ports` - (Optional, block) See [Ports](#ports) below for details.
* `tags` - (Optional) List of tags to assign to the node cluster.
* `container_count` - (Optional) Number of containers to start.
* `redeploy_on_change` - (Optional) When a non-destructive config update is applied to the
  service, redeploy the running containers with the new configuration (default: `false`).

<a id="ports"></a>
### Ports

`ports` is a block within the configuration that can be repeated to specify
the port mappings of the container. Each `ports` block supports
the following:

* `internal` - (Required, int) Port within the container.
* `external` - (Required, int) Port exposed out of the container.
* `protocol` - (Optional, string) Protocol that can be used over this port,
  defaults to TCP.

## Attributes Reference

The following attributes are exported:

* `id` - The uuid of the service
