---
layout: "scaleway"
page_title: "Scaleway: security_group_rule"
sidebar_current: "docs-scaleway-resource-security_group_rule"
description: |-
  Manages Scaleway security group rules.
---

# scaleway\_security\_group\_rule

Provides security group rules. This allows security group rules to be created, updated and deleted.
For additional details please refer to [API documentation](https://developer.scaleway.com/#security-groups-manage-rules).

## Example Usage

```hcl
resource "scaleway_security_group" "test" {
  name        = "test"
  description = "test"
}

resource "scaleway_security_group_rule" "smtp_drop_1" {
  security_group = "${scaleway_security_group.test.id}"

  action    = "accept"
  direction = "inbound"
  ip_range  = "0.0.0.0/0"
  protocol  = "TCP"
  port      = 25
}
```

## Argument Reference

The following arguments are supported:

* `action` - (Required) action of rule (`accept`, `drop`)
* `direction` - (Required) direction of rule (`inbound`, `outbound`)
* `ip_range` - (Required) ip_range of rule
* `protocol` - (Required) protocol of rule (`ICMP`, `TCP`, `UDP`)
* `port` - (Optional) port of the rule

Fields `action`, `direction`, `ip_range`, `protocol`, `port` are editable.

## Attributes Reference

The following attributes are exported:

* `id` - id of the new resource
