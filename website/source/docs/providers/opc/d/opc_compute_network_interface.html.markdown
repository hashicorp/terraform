---
layout: "opc"
page_title: "Oracle: opc_compute_network_interface"
sidebar_current: "docs-opc-datasource-network-interface"
description: |-
  Gets information about the configuration of an instance's network interface
---

# opc\_compute\_network\_interface

Use this data source to access the configuration of an instance's network interface

## Example Usage

```hcl
data "opc_compute_network_interface" "foo" {
  instance_id   = "${opc_compute_instance.my_instance.id}"
  instance_name = "${opc_compute_instance.my_instance.name}"
  interface     = "eth0"
}

output "mac_address" {
  value = "${data.opc_compute_network_interface.foo.mac_address}"
}

output "vnic" {
  value = "${data.opc_compute_network_interface.foo.vnic}"
}
```

## Argument Reference
* `instance_name` is the name of the instance.
* `instance_id` is the id of the instance.
* `interface` is the name of the attached interface. `eth0`, `eth1`, ... `eth9`.

## Attributes Reference

* `dns` - Array of DNS servers for the interface.
* `ip_address` - IP Address assigned to the interface.
* `ip_network` - The IP Network assigned to the interface.
* `mac_address` - The MAC address of the interface.
* `model` - The model of the NIC card used.
* `name_servers` - Array of name servers for the interface.
* `nat` - The IP Reservation (in IP Networks) associated with the interface.
* `search_domains` - The search domains that are sent through DHCP as option 119.
* `sec_lists` - The security lists the interface is added to.
* `shared_network` - Whether or not the interface is inside the Shared Network or an IP Network.
* `vnic` - The name of the vNIC created for the IP Network.
* `vnic_sets` - The array of vNIC Sets the interface was added to.
