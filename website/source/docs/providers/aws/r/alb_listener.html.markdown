---
layout: "aws"
page_title: "AWS: aws_alb_listener"
sidebar_current: "docs-aws-resource-alb-listener"
description: |-
  Provides an Application Load Balancer Listener resource.
---

# aws\_alb\_listener

Provides an Application Load Balancer Listener resource.

## Example Usage

```hcl
# Create a new load balancer
resource "aws_alb" "front_end" {
  # ...
}

resource "aws_alb_target_group" "front_end" {
  # ...
}

resource "aws_alb_listener" "front_end" {
  load_balancer_arn = "${aws_alb.front_end.arn}"
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2015-05"
  certificate_arn   = "arn:aws:iam::187416307283:server-certificate/test_cert_rab3wuqwgja25ct3n4jdj2tzu4"

  default_action {
    target_group_arn = "${aws_alb_target_group.front_end.arn}"
    type             = "forward"
  }
}
```

## Argument Reference

The following arguments are supported:

* `load_balancer_arn` - (Required, Forces New Resource) The ARN of the load balancer.
* `port` - (Required) The port on which the load balancer is listening.
* `protocol` - (Optional) The protocol for connections from clients to the load balancer. Valid values are `HTTP` and `HTTPS`. Defaults to `HTTP`.
* `ssl_policy` - (Optional) The name of the SSL Policy for the listener. Required if `protocol` is `HTTPS`.
* `certificate_arn` - (Optional) The ARN of the SSL server certificate. Exactly one certificate is required if the protocol is HTTPS.
* `default_action` - (Required) An Action block. Action blocks are documented below.

Action Blocks (for `default_action`) support the following:

* `target_group_arn` - (Required) The ARN of the Target Group to which to route traffic.
* `type` - (Required) The type of routing action. The only valid value is `forward`.

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The ARN of the listener (matches `arn`)
* `arn` - The ARN of the listener (matches `id`)

## Import

Listeners can be imported using their ARN, e.g.

```
$ terraform import aws_alb_listener.front_end arn:aws:elasticloadbalancing:us-west-2:187416307283:listener/app/front-end-alb/8e4497da625e2d8a/9ab28ade35828f96
```
