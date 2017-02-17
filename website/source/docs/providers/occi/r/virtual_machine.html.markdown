---
layout: "occi"
page_title: "OCCI: occi_virtual_machine"
sidebar_current: "docs-occi-resource-virtual-machine"
description: |-
  Provides a OCCI Virtual Machine resource. This can be used to create and delete virtual machines.
---

# occi\_virtual\_machine

Provides a OCCI Virtual Machine resource. This can be used to create and delete virtual machines.

## Example Usage

```
# Create a new virtual machine
resource "occi_virtual_machine" "vm" {
  image_template = ".../os_tpl#uuid_egi_centos_7_fedcloud_warg_149"
  resource_template = ".../flavour/1.0#large"
  endpoint = "..."
  name = "test_vm_micro"
  x509 = "/tmp/x509up_u1000"
  init_file = "/home/cloud-user/context"
}
```

## Argument Reference

The following arguments are supported:

* `image_template` - (Required) VM image template
* `resource_template` - (Required) VM resource template
* `name` - (Required) VM name
* `endpoint` - (Required) OCCI endpoint site
* `x509` - (Required) The VOMS proxy file for authentication
* `init_file` - (Required) [Cloud-init](https://cloudinit.readthedocs.io/en/latest/) file
* `storage_size` - (Optional) Size of block storage in GB to link to VM.
* `network` - (Optional) Connect VM to existing network. Required for some OCCI endpoints.

## Attributes Reference

The following attributes are exported:

* `image_template` - The image template of VM
* `resource_template` - The resource template of VM
* `name`- The name of VM
* `endpoint` - The OCCI endpoint of VM
* `vm_id` - The ID of VM
* `storage_size` - The size of linked block storage in GB
* `storage_id` - The ID of linked block storage
* `ip_address` - The IPv4 address
