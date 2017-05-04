---
layout: "opc"
page_title: "Oracle: opc_compute_vnic_set"
sidebar_current: "docs-opc-resource-vnic-set"
description: |-
  Creates and manages a virtual NIC set in an OPC identity domain
---

# opc\_compute\_vnic\_set

The ``opc_compute_vnic_set`` resource creates and manages a virtual NIC set in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_vnic_set" "test_set" {
  name         = "test_vnic_set"
  description  = "My vnic set"
  applied_acls = ["acl1", "acl2"]
  virtual_nics = ["nic1", "nic2", "nic3"]
  tags         = ["xyzzy", "quux"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within this identity domain) name of the virtual nic set.

* `description` - (Optional) A description of the virtual nic set.

* `applied_acls` - (Optional) A list of the ACLs to apply to the virtual nics in the set.

* `virtual_nics` - (Optional) List of virtual NICs associated with this virtual NIC set.

* `tags` - (Optional) A list of tags to apply to the storage volume.

## Import

VNIC Set's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_vnic_set.set1 example
```
