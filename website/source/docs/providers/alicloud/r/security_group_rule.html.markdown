---
layout: "alicloud"
page_title: "Alicloud: alicloud_security_group_rule"
sidebar_current: "docs-alicloud-resource-security-group-rule"
description: |-
  Provides a Alicloud Security Group Rule resource.
---

# alicloud\_security\_group\_rule

Provides a security group rule resource.
Represents a single `ingress` or `egress` group rule, which can be added to external Security Groups.

~> **NOTE:**  `nic_type` should set to `intranet` when security group type is `vpc`. In this situation it does not distinguish between intranet and internet, the rule is effective on them both.


## Example Usage

Basic Usage

```
resource "alicloud_security_group" "default" {
  name = "default"
}

resource "alicloud_security_group_rule" "allow_all_tcp" {
  type              = "ingress"
  ip_protocol       = "tcp"
  nic_type          = "internet"
  policy            = "accept"
  port_range        = "1/65535"
  priority          = 1
  security_group_id = "${alicloud_security_group.default.id}"
  cidr_ip           = "0.0.0.0/0"
}
```

## Argument Reference

The following arguments are supported:

* `type` - (Required) The type of rule being created. Valid options are `ingress` (inbound) or `egress` (outbound).
* `ip_protocol` - (Required) The protocol. Can be `tcp`, `udp`, `icmp`, `gre` or `all`.
* `port_range` - (Required) The range of port numbers relevant to the IP protocol. When the protocol is tcp or udp, the default port number range is 1-65535. For example, `1/200` means that the range of the port numbers is 1-200.
* `security_group_id` - (Required) The security group to apply this rule to.
* `nic_type` - (Optional, Forces new resource) Network type, can be either `internet` or `intranet`, the default value is `internet`.
* `policy` - (Optional, Forces new resource) Authorization policy, can be either `accept` or `drop`, the default value is `accept`.
* `priority` - (Optional, Forces new resource) Authorization policy priority, with parameter values: `1-100`, default value: 1.
* `cidr_ip` - (Optional, Forces new resource) The target IP address range. The default value is 0.0.0.0/0 (which means no restriction will be applied). Other supported formats include 10.159.6.18/12. Only IPv4 is supported.
* `source_security_group_id` - (Optional, Forces new resource) The target security group ID within the same region. Either the `source_security_group_id` or `cidr_ip` must be set. If both are set, then `cidr_ip` is authorized by default. If this field is specified, but no `cidr_ip` is specified, the `nic_type` can only select `intranet`.
* `source_group_owner_account` - (Optional, Forces new resource) The Alibaba Cloud user account Id of the target security group when security groups are authorized across accounts.  This parameter is invalid if `cidr_ip` has already been set.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the security group rule
* `type` - The type of rule, `ingress` or `egress`
* `name` - The name of the security group
* `port_range` - The range of port numbers
* `ip_protocol` - The protocol of the security group rule