---
layout: "opc"
page_title: "Oracle: opc_compute_ip_association"
sidebar_current: "docs-opc-resource-ip-association"
description: |-
  Creates and manages an IP association in an OPC identity domain for the Shared Network.
---

# opc\_compute\_ip\_association

The ``opc_compute_ip_association`` resource creates and manages an association between an IP address and an instance in
an OPC identity domain, for the Shared Network.

## Example Usage

```hcl
resource "opc_compute_ip_association" "instance1_reservation1" {
  vcable     = "${opc_compute_instance.test_instance.vcable}"
  parentpool = "ipreservation:${opc_compute_ip_reservation.reservation1.name}"
}
```

## Argument Reference

The following arguments are supported:

* `vcable` - (Required) The vcable of the instance to associate the IP address with.

* `parentpool` - (Required) The pool from which to take an IP address. To associate a specific reserved IP address, use
the prefix `ipreservation:` followed by the name of the IP reservation. To allocate an IP address from a pool, use the
prefix `ippool:`, e.g. `ippool:/oracle/public/ippool`.


## Attributes Reference

The following attributes are exported:

* `name` The name of the IP Association

## Import

IP Associations can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_ip_association.association1 example
```
