---
layout: "profitbricks"
page_title: "ProfitBricks: profitbricks_loadbalancer"
sidebar_current: "docs-profitbricks-resource-loadbalancer"
description: |-
  Creates and manages Load Balancers
---

# profitbricks\_loadbalancer

Manages a Load Balancers on ProfitBricks

## Example Usage

```hcl
resource "profitbricks_loadbalancer" "example" {
  datacenter_id = "${profitbricks_datacenter.example.id}"
  nic_id        = "${profitbricks_nic.example.id}"
  name          = "load balancer name"
  dhcp          = true
}
```

##Argument reference

* `datacenter_id` - (Required)[string]
* `nic_id` - (Required)[string]
* `dhcp` - (Optional) [boolean] Indicates if the load balancer will reserve an IP using DHCP.
* `ip` - (Optional) [string] IPv4 address of the load balancer.

