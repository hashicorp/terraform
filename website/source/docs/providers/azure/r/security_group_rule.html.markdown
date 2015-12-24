---
layout: "azure"
page_title: "Azure: azure_security_group_rule"
sidebar_current: "docs-azure-resource-security-group-rule"
description: |-
  Creates a new network security rule to be associated with a given security group.
---

# azure\_security\_group\_rule

Creates a new network Security Group Rule to be associated with a number of
given Security Groups.

~> **NOTE on Security Group Rules**: for usability purposes; Terraform allows the
addition of a single Security Group Rule to multiple Security Groups, despite
it having to define each rule individually per Security Group on Azure. As a
result; in the event that one of the Rules on one of the Groups is modified by
external factors, Terraform cannot reason as to whether or not that change
should be propagated to the others; let alone choose one changed Rule
configuration over another in case of a conflic. As such; `terraform refresh`
only checks that the rule is still defined for each of the specified
`security_group_names`; ignoring the actual parameters of the Rule and **not**
updating the state with regards to them.

## Example Usage

```
resource "azure_security_group" "web" {
    ...
}

resource "azure_security_group" "apps" {
    ...
}

resource "azure_security_group_rule" "ssh_access" {
    name = "ssh-access-rule"
    security_group_names = ["${azure_security_group.web.name}", "${azure_security_group.apps.name}"]
    type = "Inbound"
    action = "Allow"
    priority = 200
    source_address_prefix = "100.0.0.0/32"
    source_port_range = "*"
    destination_address_prefix = "10.0.0.0/32"
    destination_port_range = "22"
    protocol = "TCP"
}
```

## Argument Reference

The following arguments are supported:
* `name` - (Required) The name of the security group rule.

* `security_group_names` - (Required) A list of the names of the security groups
    the rule should be applied to.
    Changing this list forces the creation of a new resource.

* `type` - (Required) The type of the security rule. Valid options are:
    `Inbound` and `Outbound`.

* `priority` - (Required) The priority of the network security rule. Rules with
    lower priority are evaluated first. This value can be between 100 and 4096.

* `action` - (Optional) The action that is performed when the security rule is
    matched. Valid options are: `Allow` and `Deny`.

* `source_address_prefix` - (Required) The address prefix of packet sources that
    that should be subjected to the rule. An asterisk (\*) can also be used to
    match all source IPs.

* `source_port_range` - (Required) The source port or range. This value can be
    between 0 and 65535. An asterisk (\*) can also be used to match all ports.

* `destination_address_prefix` - (Required) The address prefix of packet
    destinations that should be subjected to the rule. An asterisk
    (\*) can also be used to match all destination IPs.

* `destination_port_range` - (Required) The destination port or range. This value
    can be between 0 and 65535. An asterisk (\*) can also be used to match all
    ports.

* `protocol` - (Optional) The protocol of the security rule. Valid options are:
    `TCP`, `UDP` and `*`.

The following attributes are exported:

* `id` - The security group rule ID. Coincides with its given `name`.
