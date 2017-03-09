---
layout: "dockercloud"
page_title: "Docker Cloud: dockercloud_stack"
sidebar_current: "docs-dockercloud-resource-stack"
description: |-
  Provides a Docker Cloud stack resource.
---

# dockercloud\_stack

Provides a Docker Cloud stack resource.

## Example Usage

```
# Create a new node cluster
resource "dockercloud_node_cluster" "default" {
    name = "dev"
    node_provider = "aws"
    region = "us-east-1"
    size = "t2.micro"
}

# Create a sample web stack
resource "dockercloud_stack" "web" {
    name = "foostack"
    # Explicitly set dependency on the node cluster
    depends_on = ["dockercloud_node_cluster.default"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the stack (cannot contain underscores).
* `reuse_existing_volumes` - (Optional, bool) Reuse container volumes when redeploying.

## Attributes Reference

The following attributes are exported:

* `id` - The uuid of the service
* `uri` - The DockerCloud URI of the stack
