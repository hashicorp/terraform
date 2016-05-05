---
layout: "vsphere"
page_title: "Provider: VMware vSphere"
sidebar_current: "docs-vsphere-index"
description: |-
  The VMware vSphere provider is used to interact with the resources supported by
  VMware vSphere. The provider needs to be configured with the proper credentials
  before it can be used.
---

# VMware vSphere Provider

The VMware vSphere provider is used to interact with the resources supported by
VMware vSphere.
The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

~> **NOTE:** The VMware vSphere Provider currently represents _initial support_
and therefore may undergo significant changes as the community improves it. This
provider at this time only supports IPv4 addresses on virtual machines.

## Example Usage

```
# Configure the VMware vSphere Provider
provider "vsphere" {
  user           = "${var.vsphere_user}"
  password       = "${var.vsphere_password}"
  vsphere_server = "${var.vsphere_server}"
}

# Create a folder
resource "vsphere_folder" "frontend" {
  path = "frontend"
}

# Create a file
resource "vsphere_file" "ubuntu_disk" {
  datastore = "local"
  source_file = "/home/ubuntu/my_disks/custom_ubuntu.vmdk"
  destination_file = "/my_path/disks/custom_ubuntu.vmdk"
}

# Create a virtual machine within the folder
resource "vsphere_virtual_machine" "web" {
  name   = "terraform-web"
  folder = "${vsphere_folder.frontend.path}"
  vcpu   = 2
  memory = 4096

  network_interface {
    label = "VM Network"
  }

  disk {
    template = "centos-7"
  }
}
```

## Argument Reference

The following arguments are used to configure the VMware vSphere Provider:

* `user` - (Required) This is the username for vSphere API operations. Can also
  be specified with the `VSPHERE_USER` environment variable.
* `password` - (Required) This is the password for vSphere API operations. Can
  also be specified with the `VSPHERE_PASSWORD` environment variable.
* `vsphere_server` - (Required) This is the vCenter server name for vSphere API
  operations. Can also be specified with the `VSPHERE_SERVER` environment
  variable.
* `allow_unverified_ssl` - (Optional) Boolean that can be set to true to
  disable SSL certificate verification. This should be used with care as it
  could allow an attacker to intercept your auth token. If omitted, default
  value is `false`. Can also be specified with the `VSPHERE_ALLOW_UNVERIFIED_SSL`
  environment variable.

## Required Privileges

In order to use Terraform provider as non priviledged user, a Role within
vCenter must be assigned the following privileges:

* Datastore
   - Allocate space
   - Browse datastore
   - Low level file operations
   - Remove file
   - Update virtual machine files
   - Update virtual machine metadata

* Folder (all)
   - Create folder
   - Delete folder
   - Move folder
   - Rename folder

* Network
   - Assign network

* Resource
   - Apply recommendation
   - Assign virtual machine to resource pool

* Virtual Machine
   - Configuration (all) - for now
   - Guest Operations (all) - for now
   - Interaction (all)
   - Inventory (all)
   - Provisioning (all)

These settings were tested with [vSphere
6.0](https://pubs.vmware.com/vsphere-60/index.jsp?topic=%2Fcom.vmware.vsphere.security.doc%2FGUID-18071E9A-EED1-4968-8D51-E0B4F526FDA3.html)
and [vSphere
5.5](https://pubs.vmware.com/vsphere-55/index.jsp?topic=%2Fcom.vmware.vsphere.security.doc%2FGUID-18071E9A-EED1-4968-8D51-E0B4F526FDA3.html).
For additional information on roles and permissions, please refer to official
VMware documentation.

## Virtual Machine Customization

Guest Operating Systems can be configured using
[customizations](https://pubs.vmware.com/vsphere-50/index.jsp#com.vmware.vsphere.vm_admin.doc_50/GUID-80F3F5B5-F795-45F1-B0FA-3709978113D5.html),
in order to set things properties such as domain and hostname. This mechanism
is not compatible with all operating systems, however. A list of compatible
operating systems can be found
[here](http://partnerweb.vmware.com/programs/guestOS/guest-os-customization-matrix.pdf)

If customization is attempted on an operating system which is not supported, Terraform will
create the virtual machine, but fail with the following error message:

```
Customization of the guest operating system 'debian6_64Guest' is not
supported in this configuration. Microsoft Vista (TM) and Linux guests with
Logical Volume Manager are supported only for recent ESX host and VMware Tools
versions. Refer to vCenter documentation for supported configurations.  ```
```

In order to skip the customization step for unsupported operating systems, use
the `skip_customization` argument on the virtual machine resource.

## Acceptance Tests

The VMware vSphere provider's acceptance tests require the above provider
configuration fields to be set using the documented environment variables.

In addition, the following environment variables are used in tests, and must be
set to valid values for your VMware vSphere environment:

 * VSPHERE\_IPV4\_GATEWAY
 * VSPHERE\_IPV4\_ADDRESS
 * VSPHERE\_IPV6\_GATEWAY
 * VSPHERE\_IPV6\_ADDRESS
 * VSPHERE\_NETWORK\_LABEL
 * VSPHERE\_NETWORK\_LABEL\_DHCP
 * VSPHERE\_TEMPLATE

The following environment variables depend on your vSphere environment:

 * VSPHERE\_DATACENTER
 * VSPHERE\_CLUSTER
 * VSPHERE\_RESOURCE\_POOL
 * VSPHERE\_DATASTORE

The following additional environment variables are needed for running the
"Mount ISO as CDROM media" acceptance tests.

 * VSPHERE\_CDROM\_DATASTORE
 * VSPHERE\_CDROM\_PATH


These are used to set and verify attributes on the `vsphere_virtual_machine`
resource in tests.

Once all these variables are in place, the tests can be run like this:

```
make testacc TEST=./builtin/providers/vsphere
```


