---
layout: "aws"
page_title: "AWS: aws_network_interface_attachment"
sidebar_current: "docs-aws-resource-network-interface-attachment"
description: |-
  Attach an Elastic network interface (ENI) resource with EC2 instance.
---

# aws\_network\_interface\_attachment

Attach an Elastic network interface (ENI) resource with EC2 instance.

## Example Usage

```
resource "aws_network_interface_attachment" "test" {
    instance_id = "${aws_instance.test.id}"
	network_interface_id = "${aws_network_interface.test.id}"
	device_index = 0
}
```

## Argument Reference

The following arguments are supported:

* `instance_id` - (Required) Instance ID to attach.
* `network_interface_id` - (Required) ENI ID to attach.
* `device_index` - (Required) Network interface index (int).

## Attributes Reference

The following attributes are exported:

* `instance_id` - Instance ID.
* `network_interface_id` - Network interface ID.
* `attachment_id` - The ENI Attachment ID.
* `status` - The status of the Network Interface Attachment.
