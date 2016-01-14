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

# Create a virtual machine within the folder
resource "vsphere_virtual_machine" "web" {
  name   = "terraform_web"
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

## Acceptance Tests

The VMware vSphere provider's acceptance tests require the above provider
configuration fields to be set using the documented environment variables.

In addition, the following environment variables are used in tests, and must be set to valid values for your VMware vSphere environment:

 * VSPHERE\_NETWORK\_GATEWAY
 * VSPHERE\_NETWORK\_IP\_ADDRESS
 * VSPHERE\_NETWORK\_LABEL
 * VSPHERE\_NETWORK\_LABEL\_DHCP
 * VSPHERE\_TEMPLATE

The following environment variables depend on your vSphere environment:

 * VSPHERE\_DATACENTER
 * VSPHERE\_CLUSTER
 * VSPHERE\_RESOURCE\_POOL
 * VSPHERE\_DATASTORE


These are used to set and verify attributes on the `vsphere_virtual_machine`
resource in tests.

Once all these variables are in place, the tests can be run like this:

```
make testacc TEST=./builtin/providers/vsphere
```
