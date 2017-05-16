---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_virtual_disk"
sidebar_current: "docs-vsphere-resource-virtual-disk"
description: |-
  Provides a VMware virtual disk resource.  This can be used to create and delete virtual disks.
---

# vsphere\_virtual\_disk

Provides a VMware virtual disk resource.  This can be used to create and delete virtual disks.

## Example Usage

```hcl
resource "vsphere_virtual_disk" "myDisk" {
  size	     	= 2
  vmdk_path  	= "myDisk.vmdk"
  datacenter 	= "Datacenter"
  datastore  	= "local"
  type       	= "thin"
  adapter_type  = "lsiLogic"
}
```

## Argument Reference

The following arguments are supported:

* `size` - (Required) Size of the disk (in GB).
* `vmdk_path` - (Required) The path, including filename, of the virtual disk to be created.  This should end with '.vmdk'.
* `type` - (Optional) 'eagerZeroedThick' (the default), 'lazy', or 'thin' are supported options.
* `adapter_type` - (Optional) set adapter type, 'ide' (the default), 'lsiLogic', or 'busLogic' are supported options.
* `datacenter` - (Optional) The name of a Datacenter in which to create the disk.
* `datastore` - (Required) The name of the Datastore in which to create the disk.
