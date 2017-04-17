---
layout: "triton"
page_title: "Triton: triton_vlan"
sidebar_current: "docs-triton-resource-vlan"
description: |-
    The `triton_vlan` resource represents an VLAN for a Triton account.
---

# triton\_vlan

The `triton_vlan` resource represents an Triton VLAN. A VLAN provides a low level way to segregate and subdivide the network. Traffic on one VLAN cannot, _on its own_, reach another VLAN.

## Example Usages

### Create a VLAN

```hcl
resource "triton_vlan" "dmz" {
  vlan_id     = 100
  name        = "dmz"
  description = "DMZ VLAN"
}
```

## Argument Reference

The following arguments are supported:

* `vlan_id` - (int, Required, Change forces new resource)
    Number between 0-4095 indicating VLAN ID

* `name` - (string, Required)
    Unique name to identify VLAN

* `description` - (string, Optional)
    Description of the VLAN
