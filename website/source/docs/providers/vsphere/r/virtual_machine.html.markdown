---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_virtual_machine"
sidebar_current: "docs-vsphere-resource-virtual-machine"
description: |-
  Provides a VMware vSphere virtual machine resource. This can be used to create, modify, and delete virtual machines.
---

# vsphere\_virtual\_machine

Provides a VMware vSphere virtual machine resource. This can be used to create,
modify, and delete virtual machines.

## Example Usage

```
resource "vsphere_virtual_machine" "web" {
  name   = "terraform_web"
  vcpu   = 2
  memory = 4096

  network_interface {
    label = "VM Network"
  }

  disk {
    size = 1
    iops = 500
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The virtual machine name
* `vcpu` - (Required) The number of virtual CPUs to allocate to the virtual machine
* `memory` - (Required) The amount of RAM (in MB) to allocate to the virtual machine
* `datacenter` - (Optional) The name of a Datacenter in which to launch the virtual machine
* `cluster` - (Optional) Name of a Cluster in which to launch the virtual machine
* `resource_pool` (Optional) The name of a Resource Pool in which to launch the virtual machine
* `gateway` - (Optional) Gateway IP address to use for all network interfaces
* `domain` - (Optional) A FQDN for the virtual machine; defaults to "vsphere.local"
* `time_zone` - (Optional) The [time zone](https://www.vmware.com/support/developer/vc-sdk/visdk41pubs/ApiReference/timezone.html) to set on the virtual machine. Defaults to "Etc/UTC"
* `dns_suffixes` - (Optional) List of name resolution suffixes for the virtual network adapter
* `dns_servers` - (Optional) List of DNS servers for the virtual network adapter; defaults to 8.8.8.8, 8.8.4.4
* `network_interface` - (Required) Configures virtual network interfaces; see [Network Interfaces](#network-interfaces) below for details.
* `disk` - (Required) Configures virtual disks; see [Disks](#disks) below for details
* `boot_delay` - (Optional) Time in seconds to wait for machine network to be ready.
* `custom_configuration_parameters` - (Optional) Map of values that is set as virtual machine custom configurations.

<a id="network-interfaces"></a>
## Network Interfaces

Network interfaces support the following attributes:

* `label` - (Required) Label to assign to this network interface
* `ip_address` - (Optional) Static IP to assign to this network interface. Interface will use DHCP if this is left blank. Currently only IPv4 IP addresses are supported.
* `subnet_mask` - (Optional) Subnet mask to use when statically assigning an IP.

<a id="disks"></a>
## Disks

Disks support the following attributes:

* `template` - (Required if size not provided) Template for this disk.
* `datastore` - (Optional) Datastore for this disk
* `size` - (Required if template not provided) Size of this disk (in GB).
* `iops` - (Optional) Number of virtual iops to allocate for this disk.
