---
layout: "oracleopc"
page_title: "Oracle: opc_compute_security_association"
sidebar_current: "docs-oracleopc-resource-security-association"
description: |-
  Creates and manages a security association in an OPC identity domain.
---

# opc\_compute\_security\_association

The ``opc_compute_security_association`` resource creates and manages an association between an instance and a security
list in an OPC identity domain.

## Example Usage

```
resource "opc_compute_security_association" "test_instance_sec_list_1" {
       	vcable = "${opc_compute_instance.test_instance.vcable}"
       	seclist = "${opc_compute_security_list.sec_list1.name}"
}
```

## Argument Reference

The following arguments are supported:

* `vcable` - (Required) The `vcable` of the instance to associate to the security list.

* `seclist` - (Required) The name of the security list to associate the instance to.
