---
layout: "triton"
page_title: "Triton: triton_fabric"
sidebar_current: "docs-triton-resource-fabric"
description: |-
    The `triton_fabric` resource represents an SSH fabric for a Triton account.
---

# triton\_fabric

The `triton_fabric` resource represents an fabric for a Triton account. The fabric is a logical set of interconnected switches.

## Example Usages

### Create a fabric

```hcl
resource "triton_fabric" "dmz" {
  vlan_id            = 100
  name               = "dmz"
  description        = "DMZ Network"
  subnet             = "10.60.1.0/24"
  provision_start_ip = "10.60.1.10"
  provision_end_ip   = "10.60.1.240"
  gateway            = "10.60.1.1"
  resolvers          = ["8.8.8.8", "8.8.4.4"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (String, Required, Change forces new resource)
    Network name.

* `description` - (String, Optional, Change forces new resource)
    Optional description of network.

* `subnet` - (String, Required, Change forces new resource)
    CIDR formatted string describing network.

* `provision_start_ip` - (String, Required, Change forces new resource)
    First IP on the network that can be assigned.

* `provision_end_ip` - (String, Required, Change forces new resource)
    Last assignable IP on the network.

* `gateway` - (String, Optional, Change forces new resource)
    Optional gateway IP.

* `resolvers` - (List, Optional)
    Array of IP addresses for resolvers.

* `routes` - (Map, Optional, Change forces new resource)
    Map of CIDR block to Gateway IP address.

* `internet_nat` - (Bool, Optional, Change forces new resource)
    If a NAT zone is provisioned at Gateway IP address.

* `vlan_id` - (Int, Required, Change forces new resource)
    VLAN id the network is on. Number between 0-4095 indicating VLAN ID.

## Attribute Reference

The following attributes are exported:

* `name` - (String) - Network name.
* `public` - (Bool) - Whether or not this is an RFC1918 network.
* `fabric` - (Bool) - Whether or not this network is on a fabric.
* `description` - (String) - Optional description of network.
* `subnet` - (String) - CIDR formatted string describing network.
* `provision_start_ip` - (String) - First IP on the network that can be assigned.
* `provision_end_ip` - (String) - Last assignable IP on the network.
* `gateway` - (String) - Optional gateway IP.
* `resolvers` - (List) - Array of IP addresses for resolvers.
* `routes` - (Map) - Map of CIDR block to Gateway IP address.
* `internet_nat` - (Bool) - If a NAT zone is provisioned at Gateway IP address.
* `vlan_id` - (Int) - VLAN id the network is on. Number between 0-4095 indicating VLAN ID.
