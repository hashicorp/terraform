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
  subnet_id   = "${aws_subnet.main.id}"
  private_ips = ["10.0.0.10", "10.0.0.11"]
}

resource "aws_eip" "one" {
  vpc                       = true
  network_interface         = "${aws_network_interface.multi-ip.id}"
  associate_with_private_ip = "10.0.0.10"
}

resource "aws_eip" "two" {
  vpc                       = true
  network_interface         = "${aws_network_interface.multi-ip.id}"
  associate_with_private_ip = "10.0.0.11"
}
```

Attaching an EIP to an Instance with a pre-assigned private ip (VPC Only):

```
resource "aws_vpc" "default" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.default.id}"
}

resource "aws_subnet" "tf_test_subnet" {
  vpc_id                  = "${aws_vpc.default.id}"
  cidr_block              = "10.0.0.0/24"
  map_public_ip_on_launch = true

  depends_on = ["aws_internet_gateway.gw"]
}

resource "aws_instance" "foo" {
  # us-west-2
  ami           = "ami-5189a661"
  instance_type = "t2.micro"

  private_ip = "10.0.0.12"
  subnet_id  = "${aws_subnet.tf_test_subnet.id}"
}

resource "aws_eip" "bar" {
  vpc = true

  instance                  = "${aws_instance.foo.id}"
  associate_with_private_ip = "10.0.0.12"
}
```

## Argument Reference

The following arguments are supported:

* `vpc` - (Optional) Boolean if the EIP is in a VPC or not.
* `instance` - (Optional) EC2 instance ID.
* `network_interface` - (Optional) Network interface ID to associate with.
* `associate_with_private_ip` - (Optional) A user specified primary or secondary private IP address to
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
* `associate_with_private_ip` - Contains the user specified private IP address
(if in VPC).
* `public_ip` - Contains the public IP address.
* `instance` - Contains the ID of the attached instance.
* `network_interface` - Contains the ID of the attached network interface.

[1]: https://docs.aws.amazon.com/fr_fr/AWSEC2/latest/APIReference/API_AssociateAddress.html
