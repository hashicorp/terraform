---
layout: "azure"
page_title: "Azure: azure_security_group"
sidebar_current: "docs-azure-resource-security-group"
description: |-
  Creates a new network security group within the context of the specified subscription.
---

# azure\_security\_group

Creates a new network security group within the context of the specified
subscription.

## Example Usage

```
resource "azure_security_group" "web" {
    name = "webservers"
    location = "West US"

    rule {
        name = "HTTPS"
        priority = 101
        source_cidr = "*"
        source_port = "*"
        destination_cidr = "*"
        destination_port = "443"
        protocol = "TCP"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the security group. Changing this forces a
    new resource to be created.

* `label` - (Optional) The identifier for the security group. The label can be
    up to 1024 characters long. Changing this forces a new resource to be
    created (defaults to the security group name)

* `location` - (Required) The location/region where the security group is
    created. Changing this forces a new resource to be created.

* `rule` - (Required) Can be specified multiple times to define multiple
    rules. Each `rule` block supports fields documented below.

The `rule` block supports:

* `name` - (Required) The name of the security rule.

* `type ` - (Optional) The type of the security rule. Valid options are:
    `Inbound` and `Outbound` (defaults `Inbound`)

* `priority` - (Required) The priority of the network security rule. Rules with
    lower priority are evaluated first. This value can be between 100 and 4096.

* `action` - (Optional) The action that is performed when the security rule is
    matched. Valid options are: `Allow` and `Deny` (defaults `Allow`)

* `source_cidr` - (Required) The CIDR or source IP range. An asterisk (\*) can
    also be used to match all source IPs.

* `source_port` - (Required) The source port or range. This value can be
    between 0 and 65535. An asterisk (\*) can also be used to match all ports.

* `destination_cidr` - (Required) The CIDR or destination IP range. An asterisk
    (\*) can also be used to match all destination IPs.

* `destination_port` - (Required) The destination port or range. This value can
    be between 0 and 65535. An asterisk (\*) can also be used to match all
    ports.

* `protocol` - (Optional) The protocol of the security rule. Valid options are:
    `TCP`, `UDP` and `*` (defaults `TCP`)

## Attributes Reference

The following attributes are exported:

* `id` - The security group ID.
* `label` - The identifier for the security group.
