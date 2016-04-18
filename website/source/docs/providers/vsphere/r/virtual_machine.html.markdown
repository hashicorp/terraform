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
    template = "centos-7"
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
* `time_zone` - (Optional) The [Linux](https://www.vmware.com/support/developer/vc-sdk/visdk41pubs/ApiReference/timezone.html) or [Windows](https://msdn.microsoft.com/en-us/library/ms912391.aspx) time zone to set on the virtual machine. Defaults to "Etc/UTC"
* `dns_suffixes` - (Optional) List of name resolution suffixes for the virtual network adapter
* `dns_servers` - (Optional) List of DNS servers for the virtual network adapter; defaults to 8.8.8.8, 8.8.4.4
* `network_interface` - (Required) Configures virtual network interfaces; see [Network Interfaces](#network-interfaces) below for details.
* `disk` - (Required) Configures virtual disks; see [Disks](#disks) below for details
* `boot_delay` - (Optional) Time in seconds to wait for machine network to be ready.
* `windows_opt_config` - (Optional) Extra options for clones of Windows machines.
* `linked_clone` - (Optional) Specifies if the new machine is a [linked clone](https://www.vmware.com/support/ws5/doc/ws_clone_overview.html#wp1036396) of another machine or not.
* `custom_configuration_parameters` - (Optional) Map of values that is set as virtual machine custom configurations.

The `network_interface` block supports:

* `label` - (Required) Label to assign to this network interface
* `ipv4_address` - (Optional) Static IP to assign to this network interface. Interface will use DHCP if this is left blank. Currently only IPv4 IP addresses are supported.
* `ipv4_prefix_length` - (Optional) prefix length to use when statically assigning an IP.

The following arguments are maintained for backwards compatibility and may be
removed in a future version:

* `ip_address` - __Deprecated, please use `ipv4_address` instead_.
* `subnet_mask` - __Deprecated, please use `ipv4_prefix_length` instead_.

The `windows_opt_config` block supports:

* `product_key` - (Optional) Serial number for new installation of Windows. This serial number is ignored if the original guest operating system was installed using a volume-licensed CD.
* `admin_password` - (Optional) The password for the new `administrator` account. Omit for passwordless admin (using `""` does not work).
* `domain` - (Optional) Domain that the new machine will be placed into. If `domain`, `domain_user`, and `domain_user_password` are not all set, all three will be ignored.
* `domain_user` - (Optional) User that is a member of the specified domain.
* `domain_user_password` - (Optional) Password for domain user, in plain text.

The `disk` block supports:

* `template` - (Required if size not provided) Template for this disk.
* `datastore` - (Optional) Datastore for this disk
* `size` - (Required if template not provided) Size of this disk (in GB).
* `iops` - (Optional) Number of virtual iops to allocate for this disk.
* `type` - (Optional) 'eager_zeroed' (the default), or 'thin' are supported options.

## Attributes Reference

The following attributes are exported:

* `id` - The instance ID.
* `name` - See Argument Reference above.
* `vcpu` - See Argument Reference above.
* `memory` - See Argument Reference above.
* `datacenter` - See Argument Reference above.
* `network_interface/label` - See Argument Reference above.
* `network_interface/ipv4_address` - See Argument Reference above.
* `network_interface/ipv4_prefix_length` - See Argument Reference above.
* `network_interface/ipv6_address` - Assigned static IPv6 address.
* `network_interface/ipv6_prefix_length` - Prefix length of assigned static IPv6 address.
