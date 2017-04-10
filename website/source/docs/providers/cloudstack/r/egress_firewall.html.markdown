---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_egress_firewall"
sidebar_current: "docs-cloudstack-resource-egress-firewall"
description: |-
  Creates egress firewall rules for a given network.
---

# cloudstack_egress_firewall

Creates egress firewall rules for a given network.

## Example Usage

```hcl
resource "cloudstack_egress_firewall" "default" {
  network_id = "6eb22f91-7454-4107-89f4-36afcdf33021"

  rule {
    cidr_list = ["10.0.0.0/8"]
    protocol  = "tcp"
    ports     = ["80", "1000-2000"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `network_id` - (Required) The network ID for which to create the egress
    firewall rules. Changing this forces a new resource to be created.

* `managed` - (Optional) USE WITH CAUTION! If enabled all the egress firewall
    rules for this network will be managed by this resource. This means it will
    delete all firewall rules that are not in your config! (defaults false)

* `rule` - (Optional) Can be specified multiple times. Each rule block supports
    fields documented below. If `managed = false` at least one rule is required!

* `parallelism` (Optional) Specifies how much rules will be created or deleted
    concurrently. (defaults 2)

The `rule` block supports:

* `cidr_list` - (Required) A CIDR list to allow access to the given ports.

* `protocol` - (Required) The name of the protocol to allow. Valid options are:
    `tcp`, `udp` and `icmp`.

* `icmp_type` - (Optional) The ICMP type to allow. This can only be specified if
    the protocol is ICMP.

* `icmp_code` - (Optional) The ICMP code to allow. This can only be specified if
    the protocol is ICMP.

* `ports` - (Optional) List of ports and/or port ranges to allow. This can only
    be specified if the protocol is TCP or UDP.

## Attributes Reference

The following attributes are exported:

* `id` - The network ID for which the egress firewall rules are created.
