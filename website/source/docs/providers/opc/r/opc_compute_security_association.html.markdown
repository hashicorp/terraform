---
layout: "opc"
page_title: "Oracle: opc_compute_security_association"
sidebar_current: "docs-opc-resource-security-association"
description: |-
  Creates and manages a security association in an OPC identity domain.
---

# opc\_compute\_security\_association

The ``opc_compute_security_association`` resource creates and manages an association between an instance and a security
list in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_security_association" "test_instance_sec_list_1" {
  name    = "association1"
  vcable  = "${opc_compute_instance.test_instance.vcable}"
  seclist = "${opc_compute_security_list.sec_list1.name}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Optional) The Name for the Security Association. If not specified, one is created automatically. Changing this forces a new resource to be created.

* `vcable` - (Required) The `vcable` of the instance to associate to the security list.

* `seclist` - (Required) The name of the security list to associate the instance to.

## Import

Security Association's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_security_association.association1 example
```
