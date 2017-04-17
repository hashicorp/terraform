---
layout: "aws"
page_title: "AWS: aws_inspector_resource_group"
sidebar_current: "docs-aws-resource-inspector-resource-group"
description: |-
  Provides a Inspector resource group.
---

# aws\_inspector\_resource\_group

Provides a Inspector resource group

## Example Usage

```hcl
resource "aws_inspector_resource_group" "bar" {
  tags {
    Name = "foo"
    Env  = "bar"
  }
}
```

## Argument Reference

The following arguments are supported:

* `tags` - (Required) The tags on your EC2 Instance.

## Attributes Reference

The following attributes are exported:

* `arn` - The resource group ARN.
