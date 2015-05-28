---
layout: "aws"
page_title: "AWS: aws_network_interface"
sidebar_current: "docs-aws-resource-network-interface"
description: |-
  Provides a resource to create an Elastic Network Interface
---

# aws\_network\_interface

Provides a resource to create an Elastic Network Interface.

## Example Usage

```
resource "aws_network_interface" "bar" {
    subnet_id = "${aws_subnet.foo.id}"
    private_ips = ["172.16.10.100"]
    security_groups = ["${aws_security_group.foo.id}"]

    tags {
        Name = "bar_interface"
    }
}
```

## Argument Reference

The following arguments are supported:

* `subnet_id` - (Required) The subnet ID to create in.
* `security_groups` - (Optional) A list of security group names to associate with.
* `private_ips` - (Optional) The private IP of the interface.
* `attachment` - (Optional) A mapping to attach the interface to an instance. 
* `tags` - (Optional) A mapping of tags to assign to the resource.

The `attachment` mapping supports the following:

* `instance` - (Required) The instance ID to attach to.
* `device_index` - (Required) The integer index number of the device for the network interface attachment.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Elastic Network Interface.

You can find more technical documentation about Elastic Network Interfaces in the
official [AWS User Guide](http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-eni.html).
