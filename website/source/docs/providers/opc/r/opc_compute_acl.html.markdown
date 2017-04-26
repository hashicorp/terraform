---
layout: "opc"
page_title: "Oracle: opc_compute_acl"
sidebar_current: "docs-opc-resource-acl"
description: |-
  Creates and manages an ACL in an OPC identity domain.
---

# opc\_compute\_acl

The ``opc_compute_acl`` resource creates and manages an ACL in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_acl" "default" {
  name        = "ACL1"
  description = "This is a description for an acl"
  tags        = ["tag1", "tag2"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the ACL.

* `enabled` - (Optional) Enables or disables the ACL. Set to true by default.

* `description` - (Optional) A description of the ACL.

* `tags` - (Optional) List of tags that may be applied to the ACL.

In addition to the above, the following values are exported:

* `uri` - The Uniform Resource Identifier for the ACL

## Import

ACL's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_acl.acl1 example
```
