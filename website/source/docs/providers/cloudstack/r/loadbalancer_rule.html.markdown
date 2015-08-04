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
  ipaddress = "192.168.0.1"
  algorithm = "roundrobin"
  private_port = 80
  public_port = 80
  members = ["server-1", "server-2"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the loadbalancer rule.
    Changing this forces a new resource to be created.

* `description` - (Optional) The description of the load balancer rule.

* `ipaddress` - (Required) Public ip address from where the network traffic will be load balanced from.
    Changing this forces a new resource to be created.

* `network` - (Optional) The guest network this rule will be created for. Required when public Ip address is
    not associated with any Guest network yet (VPC case).

* `algorithm` - (Required) Load balancer rule algorithm (source, roundrobin, leastconn).Changing this forces
    a new resource to be created.

* `private_port` - (Required) The private port of the private ip address/virtual machine where the network
    traffic will be load balanced to. Changing this forces a new resource to be created.

* `public_port` - (Required) The public port from where the network traffic will be load balanced from.
    Changing this forces a new resource to be created.

* `members` - (Required) List of instances to assign to the load balancer rule. Changing this forces a new
    resource to be created.

## Attributes Reference

The following attributes are exported:

* `id` - The load balancer rule ID.
* `description` - The description of the load balancer rule.
