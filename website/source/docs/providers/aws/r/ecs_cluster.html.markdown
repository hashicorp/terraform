---
layout: "aws"
page_title: "AWS: aws_ecs_cluster"
sidebar_current: "docs-aws-resource-ecs-cluster"
description: |-
  Provides an ECS cluster.
---

# aws\_ecs\_cluster

Provides an ECS cluster.

## Example Usage

```hcl
resource "aws_ecs_cluster" "foo" {
  name = "white-hart"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the cluster (up to 255 letters, numbers, hyphens, and underscores)

## Attributes Reference

The following attributes are exported:

* `name` - The name of the cluster
* `id` - The Amazon Resource Name (ARN) that identifies the cluster
