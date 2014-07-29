---
layout: "aws"
page_title: "AWS: aws_eip"
sidebar_current: "docs-aws-resource-eip"
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

## Attributes Reference

The following attributes are exported:

* `private_ip` - Contains the private IP address (if in VPC).
* `public_ip` - Contains the public IP address.
* `instance` - Contains the ID of the attached instance.

