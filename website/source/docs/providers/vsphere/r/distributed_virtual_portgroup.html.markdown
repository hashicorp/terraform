---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_dvs_portgroup"
sidebar_current: "docs-vsphere-resource-dvs_portgroup"
description: |-
  Provides a VMware vSphere Distributed Virtual Portgroup resource. This is used to provide network grouping and isolation for VMs in a Distributed Virtual Switch.
---

# vsphere\_dvs\_port\_group

  Provides a VMware vSphere Distributed Virtual Portgroup resource. This is used to provide network grouping and isolation for VMs in a Distributed Virtual Switch.

## Example Usage

```
resource "vsphere_dvs_port_group" "portgroup_1" {
  name = "portgroup_1"

  default_vlan = 10
  datacenter   = "vsphere-dc"
  type         = "earlyBinding"
  description  = "Example DVPG"
  num_ports    = 8
  auto_expand  = true
  switch_id    = "${vsphere_dvs.switch_1.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Name of the portgroup
* `datacenter` - (Required) Datacenter in which the portgroup is to be created
* `switch_id` - (Required) Switch this portgroup will belong to
* `default_vlan` - Single VLAN this portgroup is bound to
* `vlan_range` - (Repeatable) VLAN range this portgroup is bound to (fields: start, end)
* `type` - (Required) Type of portgroup (earlyBinding|ephemeral)
* `description` - Description of this portgroup
* `auto_expand` - Should this portgroup auto-expand itself (default: true)
* `num_ports` - Minimal number of ports this portgroup has 
* `port_name_format` - Format for port naming.
* `policy` - Security policy for ports in the portgroup. Fields: allow\_block\_override, allow\_live\_port\_moving, allow\_network\_resources\_pool\_override, port\_config\_reset\_disconnect, allow\_shaping\_override, allow\_traffic\_filter\_override, allow\_vendor\_config\_override.
