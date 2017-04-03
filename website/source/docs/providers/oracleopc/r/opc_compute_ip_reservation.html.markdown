---
layout: "oracleopc"
page_title: "Oracle: opc_compute_ip_reservation"
sidebar_current: "docs-oracleopc-resource-ip-reservation"
description: |-
  Creates and manages an IP reservation in an OPC identity domain.
---

# opc\_compute\_ip\_reservation

The ``opc_compute_ip_reservation`` resource creates and manages an IP reservation in an OPC identity domain.

## Example Usage

```
resource "opc_compute_ip_reservation" "reservation1" {
        parentpool = "/oracle/public/ippool"
        permanent = true
       	tags = []
}
```

## Argument Reference

The following arguments are supported:

* `parentpool` - (Required) The pool from which to allocate the IP address.

* `permanent` - (Required) Whether the IP address remains reserved even when it is no longer associated with an instance
(if true), or may be returned to the pool and replaced with a different IP address when an instance is restarted, or
deleted and recreated (if false).

* `tags` - (Optional) List of tags that may be applied to the IP reservation.
