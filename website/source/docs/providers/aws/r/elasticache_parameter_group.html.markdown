---
layout: "aws"
page_title: "AWS: aws_elasticache_parameter_group"
sidebar_current: "docs-aws-resource-elasticache-parameter-group"
---

# aws\_elasticache\_parameter\_group

Provides an ElastiCache parameter group resource.

## Example Usage

```hcl
resource "aws_elasticache_parameter_group" "default" {
  name   = "cache-params"
  family = "redis2.8"

  parameter {
    name  = "activerehashing"
    value = "yes"
  }

  parameter {
    name  = "min-slaves-to-write"
    value = "2"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ElastiCache parameter group.
* `family` - (Required) The family of the ElastiCache parameter group.
* `description` - (Optional) The description of the ElastiCache parameter group. Defaults to "Managed by Terraform".
* `parameter` - (Optional) A list of ElastiCache parameters to apply.

Parameter blocks support the following:

* `name` - (Required) The name of the ElastiCache parameter.
* `value` - (Required) The value of the ElastiCache parameter.

## Attributes Reference

The following attributes are exported:

* `id` - The ElastiCache parameter group name.


## Import

ElastiCache Parameter Groups can be imported using the `name`, e.g.

```
$ terraform import aws_elasticache_parameter_group.default redis-params
```