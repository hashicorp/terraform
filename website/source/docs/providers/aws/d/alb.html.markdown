---
layout: "aws"
page_title: "AWS: aws_alb"
sidebar_current: "docs-aws-datasource-alb-x"
description: |-
  Provides an Application Load Balancer data source.
---

# aws\_alb

Provides information about an Application Load Balancer.

This data source can prove useful when a module accepts an ALB as an input
variable and needs to, for example, determine the security groups associated
with it, etc.

## Example Usage

```hcl
variable "alb_arn" {
  type    = "string"
  default = ""
}

variable "alb_name" {
  type    = "string"
  default = ""
}

data "aws_alb" "test" {
  arn  = "${var.alb_arn}"
  name = "${var.alb_arn}"
}
```

## Argument Reference

The following arguments are supported:

* `arn` - (Optional) The full ARN of the load balancer.
* `name` - (Optional) The unique name of the load balancer.

~> **NOTE**: When both `arn` and `name` are specified, `arn` takes precedence.

## Attributes Reference

See the [ALB Resource](/docs/providers/aws/r/alb.html) for details on the
returned attributes - they are identical.
