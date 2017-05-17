---
layout: "aws"
page_title: "AWS: aws_lb_cookie_stickiness_policy"
sidebar_current: "docs-aws-resource-lb-cookie-stickiness-policy"
description: |-
  Provides a load balancer cookie stickiness policy, which allows an ELB to control the sticky session lifetime of the browser.
---

# aws\_lb\_cookie\_stickiness\_policy

Provides a load balancer cookie stickiness policy, which allows an ELB to control the sticky session lifetime of the browser.

## Example Usage

```hcl
resource "aws_elb" "lb" {
  name               = "test-lb"
  availability_zones = ["us-east-1a"]

  listener {
    instance_port     = 8000
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }
}

resource "aws_lb_cookie_stickiness_policy" "foo" {
  name                     = "foo-policy"
  load_balancer            = "${aws_elb.lb.id}"
  lb_port                  = 80
  cookie_expiration_period = 600
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the stickiness policy.
* `load_balancer` - (Required) The load balancer to which the policy
  should be attached.
* `lb_port` - (Required) The load balancer port to which the policy
  should be applied. This must be an active listener on the load
balancer.
* `cookie_expiration_period` - (Optional) The time period after which
  the session cookie should be considered stale, expressed in seconds.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the policy.
* `name` - The name of the stickiness policy.
* `load_balancer` - The load balancer to which the policy is attached.
* `lb_port` - The load balancer port to which the policy is applied.
* `cookie_expiration_period` - The time period after which the session cookie is considered stale, expressed in seconds.
