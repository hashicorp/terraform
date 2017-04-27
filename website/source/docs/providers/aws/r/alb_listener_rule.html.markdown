---
layout: "aws"
page_title: "AWS: aws_alb_listener_rule"
sidebar_current: "docs-aws-resource-alb-listener-rule"
description: |-
  Provides an Application Load Balancer Listener Rule resource.
---

# aws\_alb\_listener\_rule

Provides an Application Load Balancer Listener Rule resource.

## Example Usage

```hcl
# Create a new load balancer
resource "aws_alb" "front_end" {
  # ...
}

resource "aws_alb_listener" "front_end" {
  # Other parameters
}

resource "aws_alb_listener_rule" "static" {
  listener_arn = "${aws_alb_listener.front_end.arn}"
  priority     = 100

  action {
    type             = "forward"
    target_group_arn = "${aws_alb_target_group.static.arn}"
  }

  condition {
    field  = "path-pattern"
    values = ["/static/*"]
  }
}

resource "aws_alb_listener_rule" "host_based_routing" {
  listener_arn = "${aws_alb_listener.front_end.arn}"
  priority     = 99

  action {
    type             = "forward"
    target_group_arn = "${aws_alb_target_group.static.arn}"
  }

  condition {
    field  = "host-header"
    values = ["my-service.*.terraform.io"]
  }
}

```

## Argument Reference

The following arguments are supported:

* `listener_arn` - (Required, Forces New Resource) The ARN of the listener to which to attach the rule.
* `priority` - (Required) The priority for the rule. A listener can't have multiple rules with the same priority.
* `action` - (Required) An Action block. Action blocks are documented below.
* `condition` - (Required) A Condition block. Condition blocks are documented below.

Action Blocks (for `action`) support the following:

* `target_group_arn` - (Required) The ARN of the Target Group to which to route traffic.
* `type` - (Required) The type of routing action. The only valid value is `forward`.

Condition Blocks (for `condition`) support the following:

* `field` - (Required) The name of the field. Must be one of `path-pattern` for path based routing or `host-header` for host based routing.
* `values` - (Required) The path patterns to match. A maximum of 1 can be defined.

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The ARN of the rule (matches `arn`)
* `arn` - The ARN of the rule (matches `id`)

## Import

Rules can be imported using their ARN, e.g.

```
$ terraform import aws_alb_listener_rule.front_end arn:aws:elasticloadbalancing:us-west-2:187416307283:listener-rule/app/test/8e4497da625e2d8a/9ab28ade35828f96/67b3d2d36dd7c26b
```
