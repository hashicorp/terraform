---
layout: "aws"
page_title: "AWS: aws_load_balancer_listener_policy"
sidebar_current: "docs-aws-resource-load-balancer-listener-policy"
description: |-
  Attaches a load balancer policy to an ELB Listener.
---

# aws\_elb\_load\_balancer\_listener\_policy

Attaches a load balancer policy to an ELB Listener.


## Example Usage

```hcl
resource "aws_elb" "wu-tang" {
  name               = "wu-tang"
  availability_zones = ["us-east-1a"]

  listener {
    instance_port      = 443
    instance_protocol  = "http"
    lb_port            = 443
    lb_protocol        = "https"
    ssl_certificate_id = "arn:aws:iam::000000000000:server-certificate/wu-tang.net"
  }

  tags {
    Name = "wu-tang"
  }
}

resource "aws_load_balancer_policy" "wu-tang-ssl" {
  load_balancer_name = "${aws_elb.wu-tang.name}"
  policy_name        = "wu-tang-ssl"
  policy_type_name   = "SSLNegotiationPolicyType"

  policy_attribute = {
    name  = "ECDHE-ECDSA-AES128-GCM-SHA256"
    value = "true"
  }

  policy_attribute = {
    name  = "Protocol-TLSv1.2"
    value = "true"
  }
}

resource "aws_load_balancer_listener_policy" "wu-tang-listener-policies-443" {
  load_balancer_name = "${aws_elb.wu-tang.name}"
  load_balancer_port = 443

  policy_names = [
    "${aws_load_balancer_policy.wu-tang-ssl.policy_name}",
  ]
}
```

This example shows how to customize the TLS settings of an HTTPS listener.

## Argument Reference

The following arguments are supported:

* `load_balancer_name` - (Required) The load balancer to attach the policy to.
* `load_balancer_port` - (Required) The load balancer listener port to apply the policy to.
* `policy_names` - (Required) List of Policy Names to apply to the backend server.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the policy.
* `load_balancer_name` - The load balancer on which the policy is defined.
* `load_balancer_port` - The load balancer listener port the policies are applied to
