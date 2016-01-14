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

```
resource "aws_eip" "lb" {
    instance = "${aws_instance.web.id}"
    vpc = true
}
```

## Argument Reference

The following arguments are supported:

* `vpc` - (Optional) Boolean if the EIP is in a VPC or not.
* `instance` - (Optional) EC2 instance ID.
* `network_interface` - (Optional) Network interface ID to associate with.

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


[1]: http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_AssociateAddress.html
