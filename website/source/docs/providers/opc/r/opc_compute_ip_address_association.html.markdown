---
layout: "opc"
page_title: "Oracle: opc_compute_ip_address_association"
sidebar_current: "docs-opc-resource-ip-address-association"
description: |-
  Creates and manages an IP address association in an OPC identity domain, for an IP Network.
---

# opc\_compute\_ip\_address\_association

The ``opc_compute_ip_address_association`` resource creates and manages an IP address association between an IP address reservation and a virtual NIC in an OPC identity domain, for an IP Network.

## Example Usage

```hcl
resource "opc_compute_ip_address_association" "default" {
  name                   = "PrefixSet1"
  ip_address_reservation = "${opc_compute_ip_address_reservation.default.name}"
  vnic                   = "${data.opc_compute_vnic.default.name}"
  tags                   = ["tags1", "tags2"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ip address association.

* `ip_address_reservation` - (Optional) The name of the NAT IP address reservation.

* `vnic` - (Optional) The name of the virtual NIC associated with this NAT IP reservation.

* `description` - (Optional) A description of the ip address association.

* `tags` - (Optional) List of tags that may be applied to the ip address association.

In addition to the above, the following variables are exported:

* `uri` - (Computed) The Uniform Resource Identifier of the ip address association.

## Import

IP Address Associations can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_ip_address_association.default example
```
