---
layout: "vsphere"
page_title: "Provider: vSphere"
sidebar_current: "docs-vsphere-index"
description: |-
  The vSphere provider is used to interact with the resources supported by
  vSphere. The provider needs to be configured with the proper credentials before
  it can be used.
---

# vSphere Provider

The vSphere provider is used to interact with the resources supported by vSphere.
The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

~> **NOTE:** The vSphere Provider currently represents _initial support_ and
therefore may undergo significant changes as the community improves it.

## Example Usage

```
# Configure the vSphere Provider
provider "vsphere" {
  user           = "${var.vsphere_user}"
  password       = "${var.vsphere_password}"
  vcenter_server = "${var.vsphere_vcenter_server}"
}

# Create a virtual machine
resource "vsphere_virtual_machine" "web" {
  name   = "terraform_web"
  vcpu   = 2
  memory = 4096

  network_interface {
    label = "VM Network"
  }

  disk {
    size = 1
    iops = 500
  }
}
```

## Argument Reference

The following arguments are used to configure the vSphere Provider:

* `user` - (Required) This is the username for vSphere API operations. Can also
  be specified with the `VSPHERE_USER` environment variable.
* `password` - (Required) This is the password for vSphere API operations. Can
  also be specified with the `VSPHERE_PASSWORD` environment variable.
* `vcenter_server` - (Required) This is the vCenter server name for vSphere API
  operations. Can also be specified with the `VSPHERE_VCENTER` environment
  variable.

## Acceptance Tests

The vSphere provider's acceptance tests require the above provider
configuration fields to be set using the documented environment variables.

In addition, the following environment variables are used in tests, and must be set to valid values for your vSphere environment:

 * VSPHERE\_CLUSTER
 * VSPHERE\_DATACENTER
 * VSPHERE\_DATASTORE
 * VSPHERE\_NETWORK\_GATEWAY
 * VSPHERE\_NETWORK\_IP\_ADDRESS
 * VSPHERE\_NETWORK\_LABEL
 * VSPHERE\_NETWORK\_LABEL\_DHCP
 * VSPHERE\_TEMPLATE
 * VSPHERE\_VM\_PASSWORD

These are used to set and verify attributes on the `vsphere_virtual_machine`
resource in tests.

Once all these variables are in place, the tests can be run like this:

```
make testacc TEST=./builtin/providers/vsphere
```
