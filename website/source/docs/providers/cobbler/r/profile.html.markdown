---
layout: "cobbler"
page_title: "Cobbler: cobbler_profile"
sidebar_current: "docs-cobbler-resource-profile"
description: |-
  Manages a Profile within Cobbler.
---

# cobbler\_profile

Manages a Profile within Cobbler.

## Example Usage

```
resource "cobbler_profile" "my_profile" {
  name = "/var/lib/cobbler/snippets/my_snippet"
  distro = "ubuntu-1404-x86_64"
}
```

## Argument Reference

The following arguments are supported:

* `boot_files` - (Optional) Files copied into tftpboot beyond the
  kernel/initrd.

* `comment` - (Optional) Free form text description.

* `distro` - (Optional) Parent distribution.

* `enable_gpxe` - (Optional) Use gPXE instead of PXELINUX for
  advanced booting options.

* `enable_menu` - (Optional) Enable a boot menu.

* `fetchable_files` - (Optional) Templates for tftp or wget.

* `kernel_options` - (Optional) Kernel options for the profile.

* `kernel_options_post` - (Optional) Post install kernel options.

* `kickstart` - (Optional) The kickstart file to use.

* `ks_meta` - (Optional) Kickstart metadata.

* `mgmt_classes` - (Optional) For external configuration management.

* `mgmt_parameters` - (Optional) Parameters which will be handed to
  your management application (Must be a valid YAML dictionary).

* `name_servers_search` - (Optional) Name server search settings.

* `name_servers` - (Optional) Name servers.

* `name` - (Required) The name of the profile.

* `owners` - (Optional) Owners list for authz_ownership.

* `proxy` - (Optional) Proxy URL.

* `redhat_management_key` - (Optional) Red Hat Management Key.

* `redhat_management_server` - (Optional) RedHat Management Server.

* `repos` - (Optional) Repos to auto-assign to this profile.

* `template_files` - (Optional) File mappings for built-in config
  management.

* `template_remote_kickstarts` - (Optional) remote kickstart
  templates.

* `virt_auto_boot` - (Optional) Auto boot virtual machines.

* `virt_bridge` - (Optional) The bridge for virtual machines.

* `virt_cpus` - (Optional) The number of virtual CPUs.

* `virt_file_size` - (Optional) The virtual machine file size.

* `virt_path` - (Optional) The virtual machine path.

* `virt_ram` - (Optional) The amount of RAM for the virtual machine.

* `virt_type` - (Optional) The type of virtual machine. Valid options
  are: xenpv, xenfv, qemu, kvm, vmware, openvz.

## Attributes Reference

All of the above Optional attributes are also exported.
