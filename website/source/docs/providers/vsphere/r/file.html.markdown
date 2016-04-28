---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_file"
sidebar_current: "docs-vsphere-resource-file"
description: |-
  Provides a VMware vSphere virtual machine file resource. This can be used to files (e.g. vmdk disks) from Terraform host machine to remote vSphere.
-----------------------------------------------------------------------------------------------------------------------------------------------------

# vsphere\_file

Provides a VMware vSphere virtual machine file resource. This can be used to files (e.g. vmdk disks) from Terraform host machine to remote vSphere.

## Example Usage

```
resource "vsphere_file" "ubuntu_disk" {
  datastore = "local"
  source_file = "/home/ubuntu/my_disks/custom_ubuntu.vmdk"
  destination_file = "/my_path/disks/custom_ubuntu.vmdk"
}
```

## Argument Reference

The following arguments are supported:

* `source_file` - (Required) The path to the file on Terraform host that will be uploaded to vSphere.
* `destination_file` - (Required) The path to where the file should be uploaded to on vSphere.
* `datacenter` - (Optional) The name of a Datacenter in which the file will be created/uploaded to.
* `datastore` - (Required) The name of the Datastore in which to create/upload the file to.
