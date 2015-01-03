---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_firewall"
sidebar_current: "docs-cloudstack-resource-firewall"
description: |-
  Creates firewall rules for a given IP address.
---

# cloudstack\_firewall

Creates firewall rules for a given IP address.

## Example Usage

```
resource "cloudstack_firewall" "default" {
  ipaddress = "192.168.0.1"

  rule {
    source_cidr = "10.0.0.0/8"
    protocol = "tcp"
    ports = ["80", "1000-2000"]
  }
}
```

## Argument Reference

The following arguments are supported:

* `ipaddress` - (Required) The IP address for which to create the firewall rules.
    Changing this forces a new resource to be created.

* `rule` - (Required) Can be specified multiple times. Each rule block supports
    fields documented below.

The `rule` block supports:

* `source_cidr` - (Required) The source CIDR to allow access to the given ports.

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

* `ipaddress` - The IP address for which the firewall rules are created.
