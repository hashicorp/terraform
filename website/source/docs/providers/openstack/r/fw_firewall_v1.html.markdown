---
layout: "openstack"
page_title: "OpenStack: openstack_fw_firewall_v1"
sidebar_current: "docs-openstack-resource-fw-firewall-1"
description: |-
  Manages a v1 firewall resource within OpenStack.
---

# openstack\_fw\_firewall_v1

Manages a v1 firewall resource within OpenStack.

## Example Usage

```
resource "openstack_fw_firewall_v1" "firewall_1" {
  region = ""
  name = "my-firewall"
  policy_id = "${openstack_fw_policy_v1.policy_1.id}"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the v1 networking client.
    A networking client is needed to create a firewall. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    firewall.

* `policy_id` - (Required) The policy resource id for the firewall. Changing
    this updates the `policy_id` of an existing firewall.

* `name` - (Optional) A name for the firewall. Changing this
    updates the `name` of an existing firewall.

* `description` - (Required) A description for the firewall. Changing this
    updates the `description` of an existing firewall.

* `admin_state_up` - (Optional) Administrative up/down status for the firewall
    (must be "true" or "false" if provided - defaults to "true").
    Changing this updates the `admin_state_up` of an existing firewall.

* `tenant_id` - (Optional) The owner of the floating IP. Required if admin wants
    to create a firewall for another tenant. Changing this creates a new
    firewall.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `policy_id` - See Argument Reference above.
* `name` - See Argument Reference above.
* `description` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
