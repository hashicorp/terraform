---
layout: "aws"
page_title: "AWS: aws_alb_target_group"
sidebar_current: "docs-aws-resource-alb-target-group"
description: |-
  Provides a Target Group resource for use with Application Load
  Balancers.
---

# aws\_alb\_target\_group

Provides a Target Group resource for use with Application Load Balancer
resources.

## Example Usage

```hcl
resource "aws_alb_target_group" "test" {
  name     = "tf-example-alb-tg"
  port     = 80
  protocol = "HTTP"
  vpc_id   = "${aws_vpc.main.id}"
}

resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional, Forces new resource) The name of the target group. If omitted, Terraform will assign a random, unique name.
* `name_prefix` - (Optional, Forces new resource) Creates a unique name beginning with the specified prefix. Conflicts with `name`.
* `port` - (Required) The port on which targets receive traffic, unless overridden when registering a specific target.
* `protocol` - (Required) The protocol to use for routing traffic to the targets.
* `vpc_id` - (Required) The identifier of the VPC in which to create the target group.
* `deregistration_delay` - (Optional) The amount time for Elastic Load Balancing to wait before changing the state of a deregistering target from draining to unused. The range is 0-3600 seconds. The default value is 300 seconds.
* `stickiness` - (Optional) A Stickiness block. Stickiness blocks are documented below.
* `health_check` - (Optional) A Health Check block. Health Check blocks are documented below.
* `tags` - (Optional) A mapping of tags to assign to the resource.

Stickiness Blocks (`stickiness`) support the following:

* `type` - (Required) The type of sticky sessions. The only current possible value is `lb_cookie`.
* `cookie_duration` - (Optional) The time period, in seconds, during which requests from a client should be routed to the same target. After this time period expires, the load balancer-generated cookie is considered stale. The range is 1 second to 1 week (604800 seconds). The default value is 1 day (86400 seconds).
* `enabled` - (Optional) Boolean to enable / disable `stickiness`. Default is `true`

Health Check Blocks (`health_check`) support the following:

* `interval` - (Optional) The approximate amount of time, in seconds, between health checks of an individual target. Minimum value 5 seconds, Maximum value 300 seconds. Default 30 seconds.
* `path` - (Optional) The destination for the health check request. Default `/`.
* `port` - (Optional) The port to use to connect with the target. Valid values are either ports 1-65536, or `traffic-port`. Defaults to `traffic-port`.
* `protocol` - (Optional) The protocol to use to connect with the target. Defaults to `HTTP`.
* `timeout` - (Optional) The amount of time, in seconds, during which no response means a failed health check. Defaults to 5 seconds.
* `healthy_threshold` - (Optional) The number of consecutive health checks successes required before considering an unhealthy target healthy. Defaults to 5.
* `unhealthy_threshold` - (Optional) The number of consecutive health check failures required before considering the target unhealthy. Defaults to 2.
* `matcher` (Optional) The HTTP codes to use when checking for a successful response from a target. Defaults to `200`. You can specify multiple values (for example, "200,202") or a range of values (for example, "200-299").

## Attributes Reference

The following attributes are exported in addition to the arguments listed above:

* `id` - The ARN of the Target Group (matches `arn`)
* `arn` - The ARN of the Target Group (matches `id`)
* `arn_suffix` - The ARN suffix for use with CloudWatch Metrics.

## Import

Target Groups can be imported using their ARN, e.g.

```
$ terraform import aws_alb_target_group.app_front_end arn:aws:elasticloadbalancing:us-west-2:187416307283:targetgroup/app-front-end/20cfe21448b66314
```
