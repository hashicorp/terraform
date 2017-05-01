---
layout: "opc"
page_title: "Oracle: opc_compute_security_list"
sidebar_current: "docs-opc-resource-security-list"
description: |-
  Creates and manages a security list in an OPC identity domain.
---

# opc\_compute\_security\_list

The ``opc_compute_security_list`` resource creates and manages a security list in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_security_list" "sec_list1" {
  name                 = "sec-list-1"
  policy               = "permit"
  outbound_cidr_policy = "deny"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique (within the identity domain) name of the security list.

* `policy` - (Required) The policy to apply to instances associated with this list. Must be one of `permit`,
`reject` (packets are dropped but a reply is sent) and `deny` (packets are dropped and no reply is sent).

* `output_cidr_policy` - (Required) The policy for outbound traffic from the security list. Must be one of `permit`,
`reject` (packets are dropped but a reply is sent) and `deny` (packets are dropped and no reply is sent).

## Import

Security List's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_security_list.list1 example
```
