---
layout: "aws"
page_title: "AWS: aws_load_balancer_policy"
sidebar_current: "docs-aws-resource-load-balancer-policy"
description: |-
  Provides a load balancer policy, which can be attached to an ELB listener or backend server.
---

# aws\_elb\_load\_balancer\_policy

Provides a load balancer policy, which can be attached to an ELB listener or backend server.

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

resource "aws_load_balancer_policy" "wu-tang-ca-pubkey-policy" {
  load_balancer_name = "${aws_elb.wu-tang.name}"
  policy_name        = "wu-tang-ca-pubkey-policy"
  policy_type_name   = "PublicKeyPolicyType"

  policy_attribute = {
    name  = "PublicKey"
    value = "${file("wu-tang-pubkey")}"
  }
}

resource "aws_load_balancer_policy" "wu-tang-root-ca-backend-auth-policy" {
  load_balancer_name = "${aws_elb.wu-tang.name}"
  policy_name        = "wu-tang-root-ca-backend-auth-policy"
  policy_type_name   = "BackendServerAuthenticationPolicyType"

  policy_attribute = {
    name  = "PublicKeyPolicyName"
    value = "${aws_load_balancer_policy.wu-tang-root-ca-pubkey-policy.policy_name}"
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

resource "aws_load_balancer_backend_server_policy" "wu-tang-backend-auth-policies-443" {
  load_balancer_name = "${aws_elb.wu-tang.name}"
  instance_port      = 443

  policy_names = [
    "${aws_load_balancer_policy.wu-tang-root-ca-backend-auth-policy.policy_name}",
  ]
}

resource "aws_load_balancer_listener_policy" "wu-tang-listener-policies-443" {
  load_balancer_name = "${aws_elb.wu-tang.name}"
  load_balancer_port = 443

  policy_names = [
    "${aws_load_balancer_policy.wu-tang-ssl.policy_name}",
  ]
}
```

Where the file `pubkey` in the current directory contains only the _public key_ of the certificate.

```shell
cat wu-tang-ca.pem | openssl x509 -pubkey -noout | grep -v '\-\-\-\-' | tr -d '\n' > wu-tang-pubkey
```

This example shows how to enable backend authentication for an ELB as well as customize the TLS settings.

## Argument Reference

The following arguments are supported:

* `load_balancer_name` - (Required) The load balancer on which the policy is defined.
* `policy_name` - (Required) The name of the load balancer policy.
* `policy_type_name` - (Required) The policy type.
* `policy_attribute` - (Optional) Policy attribute to apply to the policy.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the policy.
* `policy_name` - The name of the stickiness policy.
* `policy_type_name` - The policy type of the policy.
* `load_balancer_name` - The load balancer on which the policy is defined.
