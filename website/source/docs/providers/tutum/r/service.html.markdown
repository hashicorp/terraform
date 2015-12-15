---
layout: "tutum"
page_title: "Tutum: tutum_service"
sidebar_current: "docs-tutum-resource-service"
description: |-
  Provides a Tutum service resource.
---

# tutum\_service

Provides a Tutum service resource.

## Example Usage

```
# Create a new node cluster
resource "tutum_node_cluster" "default" {
    name = "dev"
    node_provider = "aws"
    region = "us-east-1"
    size = "t2.micro"
}

# Create a sample web service
resource "tutum_service" "web" {
    name = "web_server"
    image = "python:3.2"
    entrypoint = "python -m http.server"

    # Explicitly set dependency on the node cluster
    depends_on = ["tutum_node_cluster.default"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the service (cannot contain underscores).
* `image` - (Required) The image to start the container with.
* `entrypoint` - (Optional) The entrypoint used at container startup.
* `container_count` - (Optional) Number of containers to start.
* `redeploy_on_change` - (Optional) When a non-destructive config update is applied to the
  service, redeploy the running containers with the new configuration (default: `false`).
* `tags` - (Optional) List of tags to assign to the node cluster.

## Attributes Reference

The following attributes are exported:

* `id` - The uuid of the service
