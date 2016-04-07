---
layout: "aws"
page_title: "AWS: aws_eip"
sidebar_current: "docs-aws-resource-eip"
description: |-
  Provides an Elastic IP resource.
---

# aws\_eip

Provides an Elastic IP resource.

## Example Usage

Single EIP associated with an instance:

```
resource "aws_eip" "lb" {
  instance = "${aws_instance.web.id}"
  vpc      = true
}
```

Muliple EIPs associated with a single network interface:

```
resource "aws_network_interface" "multi-ip" {
  subnet_id       = "${aws_subnet.main.id}"
	private_ips     = ["10.0.0.10", "10.0.0.11"]
}
resource "aws_eip" "one" {
	vpc               = true
	network_interface = "${aws_network_interface.multi-ip.id}"
	private_ip        = "10.0.0.10"
}
resource "aws_eip" "two" {
	vpc               = true
	network_interface = "${aws_network_interface.multi-ip.id}"
	private_ip        = "10.0.0.11"
}
```

## Argument Reference

The following arguments are supported:

* `vpc` - (Optional) Boolean if the EIP is in a VPC or not.
* `instance` - (Optional) EC2 instance ID.
* `network_interface` - (Optional) Network interface ID to associate with.
* `private_ip` - (Optional) The primary or secondary private IP address to
  associate with the Elastic IP address. If no private IP address is specified,
  the Elastic IP address is associated with the primary private IP address.

~> **NOTE:** You can specify either the `instance` ID or the `network_interface` ID,
but not both. Including both will **not** return an error from the AWS API, but will
have undefined behavior. See the relevant [AssociateAddress API Call][1] for
more information.

## Attributes Reference

The following attributes are exported:

* `id` - Contains the EIP allocation ID.
* `private_ip` - Contains the private IP address (if in VPC).
* `public_ip` - Contains the public IP address.
* `instance` - Contains the ID of the attached instance.
* `network_interface` - Contains the ID of the attached network interface.

[1]: https://docs.aws.amazon.com/fr_fr/AWSEC2/latest/APIReference/API_AssociateAddress.html
