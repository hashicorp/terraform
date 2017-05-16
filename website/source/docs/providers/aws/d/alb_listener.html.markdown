---
layout: "aws"
page_title: "AWS: aws_alb_listener"
sidebar_current: "docs-aws-datasource-alb-listener"
description: |-
  Provides an Application Load Balancer Listener data source.
---

# aws\_alb\_listener

Provides information about an Application Load Balancer Listener.

This data source can prove useful when a module accepts an ALB Listener as an
input variable and needs to know the ALB it is attached to, or other
information specific to the listener in question.

## Example Usage

```hcl
variable "listener_arn" {
  type = "string"
}

data "aws_alb_listener" "listener" {
  arn = "${var.listener_arn}"
}
```

## Argument Reference

The following arguments are supported:

* `arn` - (Required) The ARN of the listener.

## Attributes Reference

See the [ALB Listener Resource](/docs/providers/aws/r/alb_listener.html) for details
on the returned attributes - they are identical.
