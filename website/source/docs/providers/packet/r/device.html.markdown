---
layout: "packet"
page_title: "Packet: packet_device"
sidebar_current: "docs-packet-resource-device"
description: |-
  Provides a Packet device resource. This can be used to create, modify, and delete devices.
---

# packet\_device

Provides a Packet device resource. This can be used to create,
modify, and delete devices.

## Example Usage

```
# Create a device and add it to cool_project
resource "packet_device" "web1" {
		hostname = "tf.coreos2"
		plan = "baremetal_1"
		facility = "ewr1"
		operating_system = "coreos_stable"
		billing_cycle = "hourly"
		project_id = "${packet_project.cool_project.id}"
}
```

## Argument Reference

The following arguments are supported:

* `hostname` - (Required) The device name
* `project_id` - (Required) The id of the project in which to create the device
* `operating_system` - (Required) The operating system slug
* `facility` - (Required) The facility in which to create the device
* `plan` - (Required) The hardware config slug
* `billing_cycle` - (Required) monthly or hourly
* `user_data` (Optional) - A string of the desired User Data for the device.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the device
* `hostname`- The hostname of the device
* `project_id`- The ID of the project the device belongs to
* `facility` - The facility the device is in
* `plan` - The hardware config of the device
* `network` - The private and public v4 and v6 IPs assigned to the device
* `locked` - Whether the device is locked
* `billing_cycle` - The billing cycle of the device (monthly or hourly)
* `operating_system` - The operating system running on the device
* `status` - The status of the device
* `created` - The timestamp for when the device was created
* `updated` - The timestamp for the last time the device was updated
