---
layout: "profitbricks"
page_title: "ProfitBricks: profitbricks_nic"
sidebar_current: "docs-profitbricks-resource-nic"
description: |-
  Creates and manages Network Interface objects.
---

# profitbricks\_nic

Manages a NICs on ProfitBricks

## Example Usage

```hcl
resource "profitbricks_nic" "example" {
  datacenter_id = "${profitbricks_datacenter.example.id}"
  server_id     = "${profitbricks_server.example.id}"
  lan           = 2
  dhcp          = true
  ip            = "${profitbricks_ipblock.example.ip}"
}
```

##Argument reference

* `datacenter_id` - (Required)[string]<sup>[1](#myfootnote1)</sup>
* `server_id` - (Required)[string]<sup>[1](#myfootnote1)</sup>
* `lan` - (Required) [integer] The LAN ID the NIC will sit on.
* `name` - (Optional) [string] The name of the LAN.
* `dhcp` - (Optional) [boolean]
* `ip` - (Optional) [string] IP assigned to the NIC.
* `firewall_active` - (Optional) [boolean] If this resource is set to true and is nested under a server resource firewall, with open SSH port, resource must be nested under the nic.
* `nat` - (Optional) [boolean] Boolean value indicating if the private IP address has outbound access to the public internet.
