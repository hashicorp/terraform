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

~> **NOTE:** The VMware vSphere Provider currently represents _alpha support_
and therefore may undergo changes as the community improves it.  As always we strive
to not introduce breaking changes.  This provider is maintained by the community,
and therefore all contributions are welcome!

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

# Create a disk image
resource "vsphere_virtual_disk" "extraStorage" {
    size = 2
    vmdk_path = "myDisk.vmdk"
    datacenter = "Datacenter"
    datastore = "local"
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


## Virtual Machine Customization

### VMware Tools

This module utilizes VMware [tools][vtools] for multiple different vm level operations.  Open VM Tools for
Linux is recommended and VMware supported Windows VMware tools is recommended.

### Guest Customizations

Guest Operating Systems can be configured using
[customizations][custom], in order to set things properties such as domain and hostname. This mechanism
is not compatible with all operating systems, however. A list of compatible
operating systems can be found [here][matrix].

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

[vtools]:https://kb.vmware.com/selfservice/search.do?cmd=displayKC&docType=kc&docTypeID=DT_KB_1_1&externalId=2004754
[custom]:https://pubs.vmware.com/vsphere-50/index.jsp#com.vmware.vsphere.vm_admin.doc_50/GUID-80F3F5B5-F795-45F1-B0FA-3709978113D5.html
[matrix]:http://partnerweb.vmware.com/programs/guestOS/guest-os-customization-matrix.pdf
