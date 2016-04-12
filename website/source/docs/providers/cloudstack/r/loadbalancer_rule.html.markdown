---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_loadbalancer_rule"
sidebar_current: "docs-cloudstack-resource-loadbalancer-rule"
description: |-
  Creates a load balancer rule.
---

# cloudstack\_loadbalancer\_rule

Creates a loadbalancer rule.

## Example Usage

```
resource "cloudstack_loadbalancer_rule" "default" {
  name = "loadbalancer-rule-1"
  description = "Loadbalancer rule 1"
  ip_address_id = "30b21801-d4b3-4174-852b-0c0f30bdbbfb"
  algorithm = "roundrobin"
  private_port = 80
  public_port = 80
  member_ids = ["f8141e2f-4e7e-4c63-9362-986c908b7ea7"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the loadbalancer rule.
    Changing this forces a new resource to be created.

* `description` - (Optional) The description of the load balancer rule.

* `ip_address_id` - (Required) Public IP address ID from where the network
    traffic will be load balanced from. Changing this forces a new resource
    to be created.

* `ipaddress` - (Required, Deprecated) Public IP address from where the
    network traffic will be load balanced from. Changing this forces a new
    resource to be created.

* `network_id` - (Optional) The network ID this rule will be created for.
    Required when public IP address is not associated with any network yet
    (VPC case).

* `network` - (Optional, Deprecated) The network this rule will be created
    for. Required when public IP address is not associated with any network
    yet (VPC case).

* `algorithm` - (Required) Load balancer rule algorithm (source, roundrobin,
    leastconn). Changing this forces a new resource to be created.

* `private_port` - (Required) The private port of the private IP address 
    (virtual machine) where the network traffic will be load balanced to.
    Changing this forces a new resource to be created.

* `public_port` - (Required) The public port from where the network traffic
    will be load balanced from. Changing this forces a new resource to be
    created.

* `member_ids` - (Required) List of instance IDs to assign to the load balancer
    rule. Changing this forces a new resource to be created.

* `members` - (Required, Deprecated) List of instances to assign to the load
    balancer rule. Changing this forces a new resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The load balancer rule ID.
* `description` - The description of the load balancer rule.
