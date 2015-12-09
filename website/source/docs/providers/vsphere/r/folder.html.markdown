---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_folder"
sidebar_current: "docs-vsphere-resource-folder"
description: |-
  Provides a VMware vSphere virtual machine folder resource. This can be used to create and delete virtual machine folders.
---

# vsphere\_virtual\_machine

Provides a VMware vSphere virtual machine folder resource. This can be used to create and delete virtual machine folders.

## Example Usage

```
resource "vsphere_folder" "web" {
  path   = "terraform_web_folder"
}
```

## Argument Reference

The following arguments are supported:

* `path` - (Required) The path of the folder to be created (relative to the datacenter root); should not begin or end with a "/"
* `datacenter` - (Optional) The name of a Datacenter in which the folder will be created
* `existing_path` - (Computed) The path of any parent folder segments which existed at the time this folder was created; on a
destroy action, the (pre-) existing path is not removed.
