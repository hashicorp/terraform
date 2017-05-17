---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_security_group_rule"
sidebar_current: "docs-cloudstack-resource-security-group-rule"
description: |-
  Authorizes and revokes both ingress and egress rulea for a given security group.
---

# cloudstack_security_group_rule

Authorizes and revokes both ingress and egress rulea for a given security group.

## Example Usage

```hcl
resource "cloudstack_security_group_rule" "web" {
  security_group_id = "e340b62b-fbc2-4081-8f67-e40455c44bce"

  rule {
    cidr_list = ["0.0.0.0/0"]
    protocol  = "tcp"
    ports     = ["80", "443"]
  }

  rule {
    cidr_list                = ["192.168.0.0/24", "192.168.1.0/25"]
    protocol                 = "tcp"
    ports                    = ["80-90", "443"]
    traffic_type             = "egress"
    user_security_group_list = ["group01", "group02"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `security_group_id` - (Required) The security group ID for which to create
    the rules. Changing this forces a new resource to be created.

* `rule` - (Required) Can be specified multiple times. Each rule block supports
    fields documented below.

The `rule` block supports:

* `cidr_list` - (Optional) A CIDR list to allow access to the given ports.

* `protocol` - (Required) The name of the protocol to allow. Valid options are:
    `tcp`, `udp`, `icmp`, `all` or a valid protocol number.

* `icmp_type` - (Optional) The ICMP type to allow, or `-1` to allow `any`. This
    can only be specified if the protocol is ICMP. (defaults 0)

* `icmp_code` - (Optional) The ICMP code to allow, or `-1` to allow `any`. This
    can only be specified if the protocol is ICMP. (defaults 0)

* `ports` - (Optional) List of ports and/or port ranges to allow. This can only
    be specified if the protocol is TCP, UDP, ALL or a valid protocol number.

* `traffic_type` - (Optional) The traffic type for the rule. Valid options are:
    `ingress` or `egress` (defaults ingress).

* `user_security_group_list` - (Optional) A list of security groups to apply
    the rules to.

## Attributes Reference

The following attributes are exported:

* `id` - The security group ID for which the rules are created.
