---
layout: "cobbler"
page_title: "Cobbler: cobbler_system"
sidebar_current: "docs-cobbler-resource-system"
description: |-
  Manages a System within Cobbler.
---

# cobbler_system

Manages a System within Cobbler.

## Example Usage

```hcl
resource "cobbler_system" "my_system" {
  name         = "my_system"
  profile      = "${cobbler_profile.my_profile.name}"
  name_servers = ["8.8.8.8", "8.8.4.4"]
  comment      = "I'm a system"

  interface {
    name        = "eth0"
    mac_address = "aa:bb:cc:dd:ee:ff"
    static      = true
    ip_address  = "1.2.3.4"
    netmask     = "255.255.255.0"
  }

  interface {
    name        = "eth1"
    mac_address = "aa:bb:cc:dd:ee:fa"
    static      = true
    ip_address  = "1.2.3.5"
    netmask     = "255.255.255.0"
  }
}
```

## Argument Reference

The following arguments are supported:

* `boot_files` - (Optional) TFTP boot files copied into tftpboot.

* `comment` - (Optional) Free form text description

* `enable_gpxe` - (Optional) Use gPXE instead of PXELINUX.

* `fetchable_files` - (Optional) Templates for tftp or wget.

* `gateway` - (Optional) Network gateway.

* `hostname` - (Optional) Hostname of the system.

* `image` - (Optional) Parent image (if no profile is used).

* `interface` - (Optional)

* `ipv6_default_device` - (Optional) IPv6 default device.

* `kernel_options` - (Optional) Kernel options.
  ex: selinux=permissive.

* `kernel_options_post` - (Optional) Kernel options (post install).

* `kickstart` - (Optional) Path to kickstart template.

* `ks_meta` - (Optional) Kickstart metadata.

* `ldap_enabled` - (Optional) Configure LDAP at next config update.

* `ldap_type` - (Optional) LDAP management type.

* `mgmt_classes` - (Optional) Management classes for external config
  management.
* `mgmt_parameters` - (Optional) Parameters which will be handed to
  your management application. Must be a valid YAML dictionary.

* `monit_enabled` - (Optional) Configure monit on this machine at
  next config update.

* `name_servers_search` - (Optional) Name servers search path.

* `name_servers` - (Optional) Name servers.

* `name` - (Required) The name of the system.

* `netboot_enabled` - (Optional) (re)Install this machine at next
  boot.

* `owners` - (Optional) Owners list for authz_ownership.

* `power_address` - (Optional) Power management address.

* `power_id` - (Optional) Usually a plug number or blade name if
  power type requires it.

* `power_pass` - (Optional) Power management password.

* `power_type` - (Optional) Power management type.

* `power_user` - (Optional) Power management user.

* `profile` - (Required) Parent profile.

* `proxy` - (Optional) Proxy URL.

* `redhat_management_key` - (Optional) Red Hat management key.

* `redhat_management_server` - (Optional) Red Hat management server.

* `status` - (Optional) System status (development, testing,
  acceptance, production).

* `template_files` - (Optional) File mappings for built-in
  configuration management.

* `template_remote_kickstarts` - (Optional) template remote
  kickstarts.

* `virt_auto_boot` - (Optional) Auto boot the VM.

* `virt_cpus` - (Optional) Number of virtual CPUs in the VM.

* `virt_disk_driver` - (Optional) The on-disk format for the
  virtualization disk.

* `virt_file_size` - (Optional) Virt file size.

* `virt_path` - (Optional) Path to the VM.

* `virt_pxe_boot` - (Optional) Use PXE to build this VM?

* `virt_ram` - (Optional) The amount of RAM for the VM.

* `virt_type` - (Optional) Virtualization technology to use: xenpv,
  xenfv, qemu, kvm, vmware, openvz.

The `interface` block supports:

* `name` - (Required) The device name of the interface. ex: eth0.

* `cnames` - (Optional) Canonical name records.

* `dhcp_tag` - (Optional) DHCP tag.

* `dns_name` - (Optional) DNS name.

* `bonding_opts` - (Optional) Options for bonded interfaces.

* `bridge_opts` - (Optional) Options for bridge interfaces.

* `gateway` - (Optional) Per-interface gateway.

* `interface_type` - (Optional) The type of interface: na, master,
  slave, bond, bond_slave, bridge, bridge_slave, bonded_bridge_slave.

* `interface_master` - (Optional) The master interface when slave.

* `ip_address` - (Optional) The IP address of the interface.

* `ipv6_address` - (Optional) The IPv6 address of the interface.

* `ipv6_mtu` - (Optional) The MTU of the IPv6 address.

* `ipv6_static_routes` - (Optional) Static routes for the IPv6
  interface.

* `ipv6_default_gateway` - (Optional) The default gateawy for the
  IPv6 address / interface.

* `mac_address` - (Optional) The MAC address of the interface.

* `management` - (Optional) Whether this interface is a management
  interface.

* `netmask` - (Optional) The IPv4 netmask of the interface.

* `static` - (Optional) Whether the interface should be static or
  DHCP.

* `static_routes` - (Optional) Static routes for the interface.

* `virt_bridge` - (Optional) The virtual bridge to attach to.

## Attribute Reference

All optional attributes listed above are also exported.
