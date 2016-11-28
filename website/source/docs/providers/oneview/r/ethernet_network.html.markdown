---
layout: "oneview"
page_title: "Oneview: ethernet_network"
sidebar_current: "docs-oneview-ethernet-network"
description: |-
  Creates an ethernet network.
---

# oneview\_ethernet\_network

Creates an ethernet network.

## Example Usage

```js
resource "oneview_ethernet_network" "default" {
  name = "test-ethernet-network"
  vlanId = 71
}
```

## Argument Reference

The following arguments are supported: 

* `name` - (Required) A unique name for the resource.

* `vlanId` - (Required) The Virtual LAN (VLAN) identification number (integer) assigned to the network. 
Changing this forces a new resource.

- - -

* `purpose` - (Optional) A description of the network's role within the logical interconnect. 
  This defaults to General.
  
* `private_network` - (Optional) When enabled, the network is configured so that all downlink (server) ports 
  connected to the network are prevented from communicating with each other within the logical interconnect.
  This defaults to false.

* `smart_link` - (Optional) When enabled, the network is configured so that, within a logical interconnect, 
  all uplinks that carry the network are monitored. This defaults to false.
  
* `ethernet_network_type` - (Optional) The type of Ethernet network. This defaults to Tagged.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are exported:

* `uri` - The URI of the created resource.

* `eTag` - Entity tag/version ID of the resource.
