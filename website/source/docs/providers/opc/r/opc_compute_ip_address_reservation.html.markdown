---
layout: "opc"
page_title: "Oracle: opc_compute_ip_address_reservation"
sidebar_current: "docs-opc-resource-ip-address-reservation"
description: |-
  Creates and manages an IP address reservation in an OPC identity domain for an IP Network.
---

# opc\_compute\_ip\_address\_reservation

The ``opc_compute_ip_address_reservation`` resource creates and manages an IP address reservation in an OPC identity domain, for an IP Network.

## Example Usage

```hcl
resource "opc_compute_ip_address_reservation" "default" {
  name            = "IPAddressReservation1"
  ip_address_pool = "public-ippool"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ip address reservation.

* `ip_address_pool` - (Required) The IP address pool from which you want to reserve an IP address. Must be either `public-ippool` or `cloud-ippool`.

* `description` - (Optional) A description of the ip address reservation.

* `tags` - (Optional) List of tags that may be applied to the IP address reservation.

In addition to the above, the following attributes are exported:

* `ip_address` - Reserved NAT IPv4 address from the IP address pool.

* `uri` - The Uniform Resource Identifier of the ip address reservation

## Import

IP Address Reservations can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_ip_address_reservation.default example
```
