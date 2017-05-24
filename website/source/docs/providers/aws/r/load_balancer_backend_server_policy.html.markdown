---
layout: "aws"
page_title: "AWS: aws_load_balancer_backend_server_policy"
sidebar_current: "docs-aws-resource-load-balancer-backend-server-policy"
description: |-
  Attaches a load balancer policy to an ELB backend server.
---

# aws\_elb\_load\_balancer\_backend\_server\_policy

Attaches a load balancer policy to an ELB backend server.


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

resource "aws_load_balancer_backend_server_policy" "wu-tang-backend-auth-policies-443" {
  load_balancer_name = "${aws_elb.wu-tang.name}"
  instance_port      = 443

  policy_names = [
    "${aws_load_balancer_policy.wu-tang-root-ca-backend-auth-policy.policy_name}",
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

* `load_balancer_name` - (Required) The load balancer to attach the policy to.
* `policy_names` - (Required) List of Policy Names to apply to the backend server.
* `instance_port` - (Required) The instance port to apply the policy to.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the policy.
* `load_balancer_name` - The load balancer on which the policy is defined.
* `instance_port` - The backend port the policies are applied to
