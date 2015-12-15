---
layout: "tutum"
page_title: "Tutum: tutum_node_cluster"
sidebar_current: "docs-tutum-resource-node-cluster"
description: |-
  Provides a Tutum node cluster resource.
---

# tutum\_node\_cluster

Provides a Tutum node cluster resource.

## Example Usage

```
# Create a new node cluster
resource "tutum_node_cluster" "default" {
    name = "dev"
    node_provider = "aws"
    region = "us-east-1"
    size = "t2.micro"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the node cluster.
* `node_provider` - (Required) The cloud provider used for instance creation.
* `region` - (Required) The region to use with your cloud provider.
* `size` - (Required) The size of the instance (cloud provider specific).
* `disk` - (Optional) Size of the instance disk in GB.
* `node_count` - (Optional) Number of instances to create in the cluster.
* `tags` - (Optional) List of tags to assign to the node cluster.

## Attributes Reference

The following attributes are exported:

* `id` - The uuid of the node cluster

