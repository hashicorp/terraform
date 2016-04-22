---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_network_acl_rule"
sidebar_current: "docs-cloudstack-resource-network-acl-rule"
description: |-
  Creates network ACL rules for a given network ACL.
---

# cloudstack\_network\_acl\_rule

Creates network ACL rules for a given network ACL.

## Example Usage

```
resource "cloudstack_network_acl_rule" "default" {
  acl_id = "f3843ce0-334c-4586-bbd3-0c2e2bc946c6"

  rule {
    action = "allow"
    cidr_list = ["10.0.0.0/8"]
    protocol = "tcp"
    ports = ["80", "1000-2000"]
    traffic_type = "ingress"
  }
}
```

## Argument Reference

The following arguments are supported:

* `acl_id` - (Required) The network ACL ID for which to create the rules.
    Changing this forces a new resource to be created.

* `aclid` - (Required, Deprecated) The network ACL ID for which to create
    the rules. Changing this forces a new resource to be created.

* `managed` - (Optional) USE WITH CAUTION! If enabled all the firewall rules for
    this network ACL will be managed by this resource. This means it will delete
    all firewall rules that are not in your config! (defaults false)

* `rule` - (Optional) Can be specified multiple times. Each rule block supports
    fields documented below. If `managed = false` at least one rule is required!

* `parallelism` (Optional) Specifies how much rules will be created or deleted
    concurrently. (defaults 2)
    
The `rule` block supports:

* `action` - (Optional) The action for the rule. Valid options are: `allow` and
    `deny` (defaults allow).

* `cidr_list` - (Required) A CIDR list to allow access to the given ports.

* `source_cidr` - (Optional, Deprecated) The source CIDR to allow access to the
    given ports. This attribute is deprecated, please use `cidr_list` instead.

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

* `id` - The ACL ID for which the rules are created.
