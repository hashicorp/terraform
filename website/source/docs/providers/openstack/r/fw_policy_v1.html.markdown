---
layout: "openstack"
page_title: "OpenStack: openstack_fw_policy_v1"
sidebar_current: "docs-openstack-resource-fw-policy-v1"
description: |-
  Manages a v1 firewall policy resource within OpenStack.
---

# openstack\_fw\_policy_v1

Manages a v1 firewall policy resource within OpenStack.

## Example Usage

```
resource "openstack_fw_rule_v1" "rule_1" {
  name = "my-rule-1"
  description = "drop TELNET traffic"
  action = "deny"
  protocol = "tcp"
  destination_port = "23"
  enabled = "true"
}

resource "openstack_fw_rule_v1" "rule_2" {
  name = "my-rule-2"
  description = "drop NTP traffic"
  action = "deny"
  protocol = "udp"
  destination_port = "123"
  enabled = "false"
}

resource "openstack_fw_policy_v1" "policy_1" {
  region = ""
  name = "my-policy"
  rules = ["${openstack_fw_rule_v1.rule_1.id}",
           "${openstack_fw_rule_v1.rule_2.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the v1 networking client.
    A networking client is needed to create a firewall policy. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    firewall policy.

* `name` - (Optional) A name for the firewall policy. Changing this
    updates the `name` of an existing firewall policy.

* `description` - (Optional) A description for the firewall policy. Changing
    this updates the `description` of an existing firewall policy.

* `rules` - (Optional) An array of one or more firewall rules that comprise
    the policy. Changing this results in adding/removing rules from the
    existing firewall policy.

* `audited` - (Optional) Audit status of the firewall policy
    (must be "true" or "false" if provided - defaults to "false").
    This status is set to "false" whenever the firewall policy or any of its
    rules are changed. Changing this updates the `audited` status of an existing
    firewall policy.

* `shared` - (Optional) Sharing status of the firewall policy (must be "true"
    or "false" if provided - defaults to "false"). If this is "true" the policy
    is visible to, and can be used in, firewalls in other tenants. Changing this
    updates the `shared` status of an existing firewall policy.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `description` - See Argument Reference above.
* `audited` - See Argument Reference above.
* `shared` - See Argument Reference above.
