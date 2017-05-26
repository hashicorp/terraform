---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_file"
sidebar_current: "docs-vsphere-resource-file"
description: |-
  Provides a VMware vSphere virtual machine file resource. This can be used to upload files (e.g. vmdk disks) from the Terraform host machine to a remote vSphere or copy fields within vSphere.
---

# vsphere\_file

Provides a VMware vSphere virtual machine file resource. This can be used to upload files (e.g. vmdk disks) from the Terraform host machine to a remote vSphere.  The file resource can also be used to copy files within vSphere.  Files can be copied between Datacenters and/or Datastores.

Updates to file resources will handle moving a file to a new destination (datacenter and/or datastore and/or destination_file).  If any source parameter (e.g. `source_datastore`, `source_datacenter` or `source_file`) are changed, this results in a new resource (new file uploaded or copied and old one being deleted).

## Example Usages

**Upload file to vSphere:**

```hcl
resource "vsphere_file" "ubuntu_disk_upload" {
  datacenter       = "my_datacenter"
  datastore        = "local"
  source_file      = "/home/ubuntu/my_disks/custom_ubuntu.vmdk"
  destination_file = "/my_path/disks/custom_ubuntu.vmdk"
}
```

**Copy file within vSphere:**

```hcl
resource "vsphere_file" "ubuntu_disk_copy" {
  source_datacenter = "my_datacenter"
  datacenter        = "my_datacenter"
  source_datastore  = "local"
  datastore         = "local"
  source_file       = "/my_path/disks/custom_ubuntu.vmdk"
  destination_file  = "/my_path/custom_ubuntu_id.vmdk"
}
```

## Argument Reference

If `source_datacenter` and `source_datastore` are not provided, the file resource will upload the file from Terraform host.  If either `source_datacenter` or `source_datastore` are provided, the file resource will copy from within specified locations in vSphere.

The following arguments are supported:

* `source_file` - (Required) The path to the file being uploaded from the Terraform host to vSphere or copied within vSphere.
* `destination_file` - (Required) The path to where the file should be uploaded or copied to on vSphere.
* `source_datacenter` - (Optional) The name of a Datacenter in which the file will be copied from.
* `datacenter` - (Optional) The name of a Datacenter in which the file will be uploaded to.
* `source_datastore` - (Optional) The name of the Datastore in which file will be copied from.
* `datastore` - (Required) The name of the Datastore in which to upload the file to.
* `create_directories` - (Optional) Create directories in `destination_file` path parameter if any missing for copy operation.  *Note: Directories are not deleted on destroy operation.
