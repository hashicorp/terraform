---
layout: "opc"
page_title: "Oracle: opc_compute_ip_reservation"
sidebar_current: "docs-opc-resource-ip-reservation"
description: |-
  Creates and manages an IP reservation in an OPC identity domain for the Shared Network.
---

# opc\_compute\_ip\_reservation

The ``opc_compute_ip_reservation`` resource creates and manages an IP reservation in an OPC identity domain for the Shared Network.

## Example Usage

```hcl
resource "opc_compute_ip_reservation" "reservation1" {
  parent_pool = "/oracle/public/ippool"
  permanent   = true
  tags        = [ "test" ]
}
```

## Argument Reference

The following arguments are supported:

* `permanent` - (Required) Whether the IP address remains reserved even when it is no longer associated with an instance
(if true), or may be returned to the pool and replaced with a different IP address when an instance is restarted, or
deleted and recreated (if false).

* `parent_pool` - (Optional) The pool from which to allocate the IP address. Defaults to `/oracle/public/ippool`, and is currently the only acceptable input.

* `name` - (Optional) Name of the IP Reservation. Will be generated if unspecified.

* `tags` - (Optional) List of tags that may be applied to the IP reservation.

## Import

IP Reservations can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_ip_reservations.reservation1 example
```
