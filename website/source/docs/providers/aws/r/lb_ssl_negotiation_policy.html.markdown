---
layout: "aws"
page_title: "AWS: aws_lb_ssl_negotiation_policy"
sidebar_current: "docs-aws-resource-lb-ssl-negotiation-policy"
description: |-
  Provides a load balancer SSL negotiation policy, which allows an ELB to control which ciphers and protocols are supported during SSL negotiations between a client and a load balancer.
---

# aws\_lb\_ssl\_negotiation\_policy

Provides a load balancer SSL negotiation policy, which allows an ELB to control the ciphers and protocols that are supported during SSL negotiations between a client and a load balancer.

## Example Usage

```hcl
resource "aws_elb" "lb" {
  name               = "test-lb"
  availability_zones = ["us-east-1a"]

  listener {
    instance_port      = 8000
    instance_protocol  = "https"
    lb_port            = 443
    lb_protocol        = "https"
    ssl_certificate_id = "arn:aws:iam::123456789012:server-certificate/certName"
  }
}

resource "aws_lb_ssl_negotiation_policy" "foo" {
  name          = "foo-policy"
  load_balancer = "${aws_elb.lb.id}"
  lb_port       = 443

  attribute {
    name  = "Protocol-TLSv1"
    value = "false"
  }

  attribute {
    name  = "Protocol-TLSv1.1"
    value = "false"
  }

  attribute {
    name  = "Protocol-TLSv1.2"
    value = "true"
  }

  attribute {
    name  = "Server-Defined-Cipher-Order"
    value = "true"
  }

  attribute {
    name  = "ECDHE-RSA-AES128-GCM-SHA256"
    value = "true"
  }

  attribute {
    name  = "AES128-GCM-SHA256"
    value = "true"
  }

  attribute {
    name  = "EDH-RSA-DES-CBC3-SHA"
    value = "false"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the SSL negotiation policy.
* `load_balancer` - (Required) The load balancer to which the policy
  should be attached.
* `lb_port` - (Required) The load balancer port to which the policy
  should be applied. This must be an active listener on the load
balancer.
* `attribute` - (Optional) An SSL Negotiation policy attribute. Each has two properties:
	* `name` - The name of the attribute
	* `value` - The value of the attribute

To set your attributes, please see the [AWS Elastic Load Balancing Developer Guide](http://docs.aws.amazon.com/ElasticLoadBalancing/latest/DeveloperGuide/elb-security-policy-table.html) for a listing of the supported SSL protocols, SSL options, and SSL ciphers.

~> **NOTE:** The AWS documentation references Server Order Preference, which the AWS Elastic Load Balancing API refers to as `Server-Defined-Cipher-Order`. If you wish to set Server Order Preference, use this value instead.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the policy.
* `name` - The name of the stickiness policy.
* `load_balancer` - The load balancer to which the policy is attached.
* `lb_port` - The load balancer port to which the policy is applied.
* `attribute` - The SSL Negotiation policy attributes.
