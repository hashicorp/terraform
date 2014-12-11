---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_network_acl_rule"
sidebar_current: "docs-cloudstack-resource-network_acl_rule"
description: |-
  Creates network ACL rules for a given network ACL.
---

# cloudstack\_network\_acl\_rule

Creates network ACL rules for a given network ACL.

## Example Usage

```
resource "cloudstack_network_acl_rule" "default" {
  aclid = "f3843ce0-334c-4586-bbd3-0c2e2bc946c6"

  rule {
    action = "allow"
    source_cidr = "10.0.0.0/8"
    protocol = "tcp"
    ports = ["80", "1000-2000"]
    traffic_type = "ingress"
  }
}
```

## Argument Reference

The following arguments are supported:

* `aclid` - (Required) The network ACL ID for which to create the rules.
    Changing this forces a new resource to be created.

* `rule` - (Required) Can be specified multiple times. Each rule block supports
    fields documented below.

The `rule` block supports:

* `action` - (Optional) The action for the rule. Valid options are: `allow` and
    `deny` (defaults allow).

* `source_cidr` - (Required) The source cidr to allow access to the given ports.

* `protocol` - (Required) The name of the protocol to allow. Valid options are:
    `tcp`, `udp`, `icmp`, `all` or a valid protocol number.

* `icmp_type` - (Optional) The ICMP type to allow. This can only be specified if
    the protocol is ICMP.

* `icmp_code` - (Optional) The ICMP code to allow. This can only be specified if
    the protocol is ICMP.

* `ports` - (Optional) List of ports and/or port ranges to allow. This can only
    be specified if the protocol is TCP, UDP, ALL or a valid protocol number.

* `traffic_type` - (Optional) The traffic type for the rule. Valid options are:
    `ingress` or `egress` (defaults ingress).

## Attributes Reference

The following attributes are exported:

* `aclid` - The ACL ID for which the rules are created.
