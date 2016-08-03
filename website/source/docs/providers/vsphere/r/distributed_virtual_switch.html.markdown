---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_dvs"
sidebar_current: "docs-vsphere-resource-dvs"
description: |-
    Provides a vSphere Distributed Virtual Switch resource. This can be used to connect VMs together.
---

# vsphere\_dvs

    Provides a vSphere Distributed Virtual Switch resource. This can be used to connect VMs together.

## Example Usage

```
resource "vsphere_dvs" "dvs_1" {
  name        = "dvs_1"
  folder      = "example_folder"
  datacenter   = "vsphere-dc"
  description = "Example Distributed Virtual Switch"
  switch_usage_policy {
    auto_upgrade_allowed = "false"
    auto_preinstall_allowed = "false"
    partial_upgrade_allowed = "false"
  }

  contact {
    name  = "J. Random BOFH <bofh@example.io>"
    infos = "Description of the contact"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The path to the file on the Terraform host that will be uploaded to vSphere.
* `folder` - (Required) The path to where the file should be uploaded to on vSphere.
* `datacenter` - (Optional) The name of a Datacenter in which the DVS is to be created.
* `extension_key` - (Optional) Extension key for the DVS.
* `description` - (Optional) description of the DVS
* `contact` - (Optional) contact informations for the manager of this DVS
* `switch_usage_policy` - switch policy
* `switch_ip_address` - IP address for the management of this field
* `num_standalone_ports` - number of ports outside of a portgroup in this DVS
