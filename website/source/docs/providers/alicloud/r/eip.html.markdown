---
layout: "alicloud"
page_title: "Alicloud: alicloud_eip"
sidebar_current: "docs-alicloud-resource-eip"
description: |-
  Provides a ECS EIP resource.
---

# alicloud\_eip

Provides a ECS EIP resource.

## Example Usage

```
# Create a new EIP.
resource "alicloud_eip" "example" {
  bandwidth            = "10"
  internet_charge_type = "PayByBandwidth"
}
```
## Argument Reference

The following arguments are supported:

* `bandwidth` - (Optional) Maximum bandwidth to the elastic public network, measured in Mbps (Mega bit per second). If this value is not specified, then automatically sets it to 5 Mbps.
* `internet_charge_type` - (Optional, Forces new resource) Internet charge type of the EIP, Valid values are `PayByBandwidth`, `PayByTraffic`. Default is `PayByBandwidth`.

## Attributes Reference

The following attributes are exported:

* `id` - The EIP ID.
* `bandwidth` - The elastic public network bandwidth.
* `internet_charge_type` - The EIP internet charge type.
* `status` - The EIP current status.
* `ip_address` - The elastic ip address
* `instance` - The ID of the instance which is associated with the EIP.
