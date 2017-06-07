---
layout: "aws"
page_title: "AWS: aws_security_group_rule"
sidebar_current: "docs-aws-resource-security-group-rule"
description: |-
  Associates a security group with a network interface.
---

# aws\_security\_group\_attachment

This resource attaches a security group to an Elastic Network Interface (ENI).
It can be used to attach a security group to any existing ENI, be it a
secondary ENI or one attached as the primary interface on an instance.

~> **NOTE on instances, interfaces, and security groups:** Terraform currently
provides the capability to assign security groups via the [`aws_instance`][1]
and the [`aws_network_interface`][2] resources. Using this resource in
conjunction with security groups provided in-line in these resources will cause
conflicts, and will lead to spurious diffs and undefined behavior - please use
one or the other.

[1]: /docs/providers/aws/d/instance.html
[2]: /docs/providers/aws/r/network_interface.html

## Example Usage

The following provides a very basic example of setting up an instance (provided
by `instance`) in the default security group, creating a security group
(provided by `sg`) and then attaching the security group to `instance`'s
primary network interface via the `aws_security_group_attachment` resource,
named `sg_attachment`:

```hcl
data "aws_ami" "ami" {
  most_recent = true

  filter {
    name   = "name"
    values = ["amzn-ami-hvm-*"]
  }

  owners = ["amazon"]
}

resource "aws_instance" "instance" {
  instance_type = "t2.micro"
  ami           = "${data.aws_ami.ami.id}"

  tags = {
    "type" = "terraform-test-instance"
  }
}

resource "aws_security_group" "sg" {
  tags = {
    "type" = "terraform-test-security-group"
  }
}

resource "aws_security_group_attachment" "sg_attachment" {
  security_group_id    = "${aws_security_group.sg.id}"
  network_interface_id = "${aws_instance.instance.primary_network_interface_id}"
}
```

In this example, `instance` is provided by the `aws_instance` data source,
fetching an external instance, possibly not managed by Terraform.
`sg_attachment` then attaches to the output instance's `network_interface_id`:

```hcl
data "aws_instance" "instance" {
  instance_id = "i-1234567890abcdef0"
}

resource "aws_security_group" "sg" {
  tags = {
    "type" = "terraform-test-security-group"
  }
}

resource "aws_security_group_attachment" "sg_attachment" {
  security_group_id    = "${aws_security_group.sg.id}"
  network_interface_id = "${data.aws_instance.instance.network_interface_id}"
}
```

## Argument Reference

 * `security_group_id` - (Required) The ID of the security group.
 * `network_interface_id` - (Required) The ID of the network interface to attach to.

## Output Reference

There are no outputs for this resource.
