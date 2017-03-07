---
layout: "aws"
page_title: "AWS: aws_directconnect_public_virtual_interface_confirm"
sidebar_current: "docs-aws-resource-directconnect-public-virtual-interface-confirm"
description: |-
  Provides a Direct Connect public Virtual Interface confirmation resource.
---

# aws\_directconnect\_public\_virtual\_interface\_confirm

Provides a Direct Connect public Virtual Interface confirmation resource.

## Example Usage

```
resource "aws_directconnect_public_virtual_interface_confirm" "vif" {
  virtual_interface_id = "dxvif-abc123"
}
```

## Argument Reference

The following arguments are supported:

* `virtual_interface_id` - (Required) The ID of the public virtual interface.
* `allow_down_state` - (Optional) .

## Attributes Reference

The following attributes are exported:

* `connection_id` - FIXME.
* `asn` - FIXME.
* `virtual_interface_name` - FIXME.
* `vlan` - FIXME.
* `amazon_address` - FIXME.
* `customer_address` - FIXME.
* `owner_account_id` - FIXME.
* `route_filter_prefixes` - FIXME.

## Import

Direct Connect public Virtual Interfaces can be imported using the `virtual_interface_id`, e.g.

```
$ terraform import aws_directconnect_public_virtual_interface_confirm.vif dxvif-abc123
```
