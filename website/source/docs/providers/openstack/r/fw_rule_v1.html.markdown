---
layout: "openstack"
page_title: "OpenStack: openstack_fw_rule_v1"
sidebar_current: "docs-openstack-resource-fw-rule-v1"
description: |-
  Manages a v1 firewall rule resource within OpenStack.
---

# openstack\_fw\_rule_v1

Manages a v1 firewall rule resource within OpenStack.

## Example Usage

```
resource "openstack_fw_rule_v1" "rule_1" {
  name = "my_rule"
  description = "drop TELNET traffic"
  action = "deny"
  protocol = "tcp"
  destination_port = "23"
  enabled = "true"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the v1 Compute client.
    A Compute client is needed to create a firewall rule. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    firewall rule.

* `name` - (Optional) A unique name for the firewall rule. Changing this
    updates the `name` of an existing firewall rule.

* `description` - (Optional) A description for the firewall rule. Changing this
    updates the `description` of an existing firewall rule.

* `protocol` - (Required) The protocol type on which the firewall rule operates.
    Changing this updates the `protocol` of an existing firewall rule.

* `action` - (Required) Action to be taken ( must be "allow" or "deny") when the
    firewall rule matches. Changing this updates the `action` of an existing
    firewall rule.

* `ip_version` - (Optional) IP version, either 4 (default) or 6. Changing this
    updates the `ip_version` of an existing firewall rule.

* `source_ip_address` - (Optional) The source IP address on which the firewall
    rule operates. Changing this updates the `source_ip_address` of an existing
    firewall rule.

* `destination_ip_address` - (Optional) The destination IP address on which the
    firewall rule operates. Changing this updates the `destination_ip_address`
    of an existing firewall rule.

* `source_port` - (Optional) The source port on which the firewall
    rule operates. Changing this updates the `source_port` of an existing
    firewall rule.

* `destination_port` - (Optional) The destination port on which the firewall
    rule operates. Changing this updates the `destination_port` of an existing
    firewall rule.

* `enabled` - (Optional) Enabled status for the firewall rule (must be "true"
    or "false" if provided - defaults to "true"). Changing this updates the
    `enabled` status of an existing firewall rule.

* `tenant_id` - (Optional) The owner of the firewall rule. Required if admin
    wants to create a firewall rule for another tenant. Changing this creates a
    new firewall rule.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `name` - See Argument Reference above.
* `description` - See Argument Reference above.
* `protocol` - See Argument Reference above.
* `action` - See Argument Reference above.
* `ip_version` - See Argument Reference above.
* `source_ip_address` - See Argument Reference above.
* `destination_ip_address` - See Argument Reference above.
* `source_port` - See Argument Reference above.
* `destination_port` - See Argument Reference above.
* `enabled` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
