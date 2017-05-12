---
layout: "azurerm"
page_title: "Azure Resource Manager: azurerm_managed_image"
sidebar_current: "docs-azurerm-resource-managed-image"
description: |-
  Create a custom virtual machine image that can be used to create virtual machines.
---

# azurerm\_managed\_image

Create a custom virtual machine image that can be used to create virtual machines.

## Example Usage Creating from VHD

```hcl
resource "azurerm_resource_group" "test" {
  name = "acctest"
  location = "West US"
}

resource "azurerm_managed_image" "test" {
  name = "acctest"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"

	os_disk {
    os_type = "Linux"
    os_state = "Generalized"
	  blob_uri = "{blob_uri}"
    size_gb = 30
	}
}
```

## Example Usage Creating from Virtual Machine (VM must be generalized beforehand)

```hcl
resource "azurerm_resource_group" "test" {
  name = "acctest"
  location = "West US"
}

resource "azurerm_managed_image" "test" {
  name = "acctest"
  location = "West US"
  resource_group_name = "${azurerm_resource_group.test.name}"
  source_virtual_machine_id = "{vm_id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the image. Changing this forces a
    new resource to be created.
* `resource_group_name` - (Required) The name of the resource group in which to create
    the managed image. Changing this forces a new resource to be created.
* `location` - (Required) Specified the supported Azure location where the resource exists.
    Changing this forces a new resource to be created.
* `source_virtual_machine_id` - (Required when creating managed image from existing VM) 
    ID of an existing VM from which the managed image
    will be created. VM must be generalized prior to creating managed image.
* `os_disk` - (Required when creating image from .vhd) The properties of the OS image that 
    will be created, documented below.
* `data_disk` - (Optional) The properties of the data images that 
    will be created, documented below. There can be multiple data_disks.
* `tags` - (Optional) A mapping of tags to assign to the resource.

`os_disk` supports the following:

* `os_type` - (Required) Specifies the type of operating system contained in the the virtual machine image. Possible values are: Windows or Linux.
* `os_state` - (Required) Specifies the state of the operating system contained in the blob. Currently, the only value is Generalized.
* `managed_disk_id` - (Optional) Specifies the ID of the managed disk resource that you want to use to create the image.
* `blob_uri` - (Optional) Specifies the URI in Azure storage of the blob that you want to use to create the image.
* `caching` - (Optional) Specifies the caching mode as 'readonly', 'readwrite', or 'none'. The default is none.
* `size_gb` - (Optional) Specifies the size of the image to be created. The target size can't be smaller than the source size.

`data_disk` supports the following:

* `lun` - (Required) Specifies the logical unit number of the data disk.
* `managed_disk_id` - (Optional) Specifies the ID of the managed disk resource that you want to use to create the image.
* `blob_uri` - (Optional) Specifies the URI in Azure storage of the blob that you want to use to create the image.
* `caching` - (Optional) Specifies the caching mode as readonly, readwrite, or none. The default is none.
* `size_gb` - (Optional) Specifies the size of the image to be created. The target size can't be smaller than the source size.

## Attributes Reference

The following attributes are exported:

* `id` - The managed image ID.

## Import

Managed Image can be imported using the `resource id`, e.g.

```
terraform import azurerm_managed_image.test /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mygroup1/providers/microsoft.compute/images/managedimage1
```
