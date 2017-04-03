---
layout: "oracleopc"
page_title: "Oracle: opc_compute_ip_association"
sidebar_current: "docs-oracleopc-resource-ip-association"
description: |-
  Creates and manages an IP association in an OPC identity domain.
---

# opc\_compute\_ip\_association

The ``opc_compute_ip_association`` resource creates and manages an association between an IP address and an instance in
an OPC identity domain.

## Example Usage

```
resource "opc_compute_ip_association" "instance1_reservation1" {
       	vcable = "${opc_compute_instance.test_instance.vcable}"
       	parentpool = "ipreservation:${opc_compute_ip_reservation.reservation1.name}"
}
```

## Argument Reference

The following arguments are supported:

* `vcable` - (Required) The vcable of the instance to associate the IP address with.

* `parentpool` - (Required) The pool from which to take an IP address. To associate a specific reserved IP address, use
the prefix `ipreservation:` followed by the name of the IP reservation. To allocate an IP address from a pool, use the
prefix `ippool:`, e.g. `ippool:/oracle/public/ippool`.
