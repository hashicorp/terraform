---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_firewall"
sidebar_current: "docs-cloudstack-resource-firewall"
description: |-
  Creates firewall rules for a given IP address.
---

# cloudstack_firewall

Creates firewall rules for a given IP address.

## Example Usage

```hcl
resource "cloudstack_firewall" "default" {
  ip_address_id = "30b21801-d4b3-4174-852b-0c0f30bdbbfb"

  rule {
    cidr_list = ["10.0.0.0/8"]
    protocol  = "tcp"
    ports     = ["80", "1000-2000"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `ip_address_id` - (Required) The IP address ID for which to create the
    firewall rules. Changing this forces a new resource to be created.

* `managed` - (Optional) USE WITH CAUTION! If enabled all the firewall rules for
    this IP address will be managed by this resource. This means it will delete
    all firewall rules that are not in your config! (defaults false)

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

* `id` - The IP address ID for which the firewall rules are created.
