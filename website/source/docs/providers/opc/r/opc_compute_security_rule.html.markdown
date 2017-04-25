---
layout: "opc"
page_title: "Oracle: opc_compute_security_rule"
sidebar_current: "docs-opc-resource-security-rule"
description: |-
  Creates and manages a security rule in an OPC identity domain.
---

# opc\_compute\_security\_rule

The ``opc_compute_security_rule`` resource creates and manages a security rule in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_security_rule" "default" {
  name               = "SecurityRule1"
  flow_direction     = "ingress"
  acl                = "${opc_compute_acl.default.name}"
  security_protocols = ["${opc_compute_security_protocol.default.name}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the security rule.

* `flow_direction` - (Required) Specify the direction of flow of traffic, which is relative to the instances, for this security rule. Allowed values are ingress or egress.

* `disabled` - (Optional) Whether to disable this security rule. This is useful if you want to temporarily disable a rule without removing it outright from your Terraform resource definition. Defaults to `false`.

* `acl` - (Optional) Name of the ACL that contains this security rule.

* `dst_ip_address_prefixes` - (Optional) List of IP address prefix set names to match the packet's destination IP address.

* `src_ip_address_prefixes` - (Optional) List of names of IP address prefix set to match the packet's source IP address.

* `dst_vnic_set` - (Optional) Name of virtual NIC set containing the packet's destination virtual NIC.

* `src_vnic_set` - (Optional) Name of virtual NIC set containing the packet's source virtual NIC.

* `security_protocols` - (Optional) List of security protocol object names to match the packet's protocol and port.

* `description` - (Optional) A description of the security rule.

* `tags` - (Optional) List of tags that may be applied to the security rule.

## Attributes Reference

In addition to the above, the following attributes are exported:

* `uri` - The Uniform Resource Identifier of the security rule.

## Import

Security Rule's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_security_rule.rule1 example
```
