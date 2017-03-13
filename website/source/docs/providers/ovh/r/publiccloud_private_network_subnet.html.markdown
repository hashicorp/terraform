---
layout: "ovh"
page_title: "OVH: publiccloud_private_network_subnet"
sidebar_current: "docs-ovh-resource-publiccloud-private-network-subnet"
description: |-
  Creates a subnet in a private network of a public cloud project.
---

# ovh_publiccloud\_private_network\_subnet

Creates a subnet in a private network of a public cloud project.

## Example Usage

```
resource "ovh_publiccloud_private_network_subnet" "subnet" {
   project_id = "67890"
   network_id = "0234543"
   region     = "GRA1"
   start      = "192.168.168.100"
   end        = "192.168.168.200"
   network    = "192.168.168.0/24"
   dhcp       = true
   no_gateway = false
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) The id of the public cloud project. If omitted,
    the `OVH_PROJECT_ID` environment variable is used.
Changing this forces a new resource to be created.

* `network_id` - (Required) The id of the network.
   Changing this forces a new resource to be created.

* `dhcp` - (Optional) Enable DHCP.
   Changing this forces a new resource to be created. Defaults to false.
_
* `start` - (Required) First ip for this region.
   Changing this value recreates the subnet.

* `end` - (Required) Last ip for this region.
   Changing this value recreates the subnet.

* `network` - (Required) Global network in CIDR format.
   Changing this value recreates the subnet

* `region` - The region in which the network subnet will be created.
   Ex.: "GRA1". Changing this value recreates the resource.

* `no_gateway` - Set to true if you don't want to set a default gateway IP.
   Changing this value recreates the resource. Defaults to false.

## Attributes Reference

The following attributes are exported:

* `project_id` - See Argument Reference above.
* `network_id` - See Argument Reference above.
* `dhcp_id` - See Argument Reference above.
* `start` - See Argument Reference above.
* `end` - See Argument Reference above.
* `network` - See Argument Reference above.
* `region` - See Argument Reference above.
* `no_gateway` - See Argument Reference above.
* `cidr` - Ip Block representing the subnet cidr.
* `ip_pools` - List of ip pools allocated in the subnet.
* `ip_pools/network` - Global network with cidr.
* `ip_pools/region` - Region where this subnet is created.
* `ip_pools/dhcp` - DHCP enabled.
* `ip_pools/end` - Last ip for this region.
* `ip_pools/start` - First ip for this region.

