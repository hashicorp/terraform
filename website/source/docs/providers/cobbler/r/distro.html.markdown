---
layout: "cobbler"
page_title: "Cobbler: cobbler_distro"
sidebar_current: "docs-cobbler-resource-distro"
description: |-
  Manages a distribution within Cobbler.
---

# cobbler_distro

Manages a distribution within Cobbler.

## Example Usage

```hcl
resource "cobbler_distro" "ubuntu-1404-x86_64" {
  name       = "foo"
  breed      = "ubuntu"
  os_version = "trusty"
  arch       = "x86_64"
  kernel     = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/linux"
  initrd     = "/var/www/cobbler/ks_mirror/Ubuntu-14.04/install/netboot/ubuntu-installer/amd64/initrd.gz"
}
```

## Argument Reference

The following arguments are supported:

* `arch` - (Required) The architecture of the distro. Valid options
  are: i386, x86_64, ia64, ppc, ppc64, s390, arm.

* `breed` - (Required) The "breed" of distribution. Valid options
  are: redhat, fedora, centos, scientific linux, suse, debian, and
  ubuntu. These choices may vary depending on the version of Cobbler
  in use.

* `boot_files` - (Optional) Files copied into tftpboot beyond the
  kernel/initrd.

* `comment` - (Optional) Free form text description.

* `fetchable_files` - (Optional) Templates for tftp or wget.

* `kernel` - (Required) Absolute path to kernel on filesystem. This
  must already exist prior to creating the distro.

* `kernel_options` - (Optional) Kernel options to use with the
  kernel.

* `kernel_options_post` - (Optional) Post install Kernel options to
  use with the kernel after installation.

* `initrd` - (Required) Absolute path to initrd on filesystem. This
  must already exist prior to creating the distro.

* `mgmt_classes` - (Optional) Management classes for external config
  management.

* `name` - (Required) A name for the distro.

* `os_version` - (Required) The version of the distro you are
  creating. This varies with the version of Cobbler you are using.
  An updated signature list may need to be obtained in order to
  support a newer version. Example: `trusty`.

* `owners` - (Optional) Owners list for authz_ownership.

* `redhat_management_key` - (Optional) Red Hat Management key.

* `redhat_management_server` - (Optional) Red Hat Management server.

* `template_files` - (Optional) File mappings for built-in config
  management.

## Attributes Reference

All of the above Optional attributes are also exported.

## Notes

The path to the `kernel` and `initrd` files must exist before
creating a Distro. Usually this involves running `cobbler import ...`
prior to creating the Distro.
