---
layout: "nsx"
page_title: "Provider: VMware NSX"
sidebar_current: "docs-nsx-index"
description: |-
  The VMware NSX provider is used to interact with the resources supported by
  VMware NSX. The provider needs to be configured with the proper credentials
  before it can be used.
---

# VMware NSX Provider

The VMware NSX provider is used to interact with the resources supported by
VMware NSX.
The provider needs to be configured with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

~> **NOTE:** The VMware NSX Provider currently represents _initial support_
and therefore may undergo significant changes as the community improves it.

## Example Usage

```hcl
# Configure the VMware NSX Provider
provider "nsx" {
    nsx_user = "username"
    nsx_password = "password"
    nsx_server = "nsx_server_address"
    insecure = True
}

# Create a service resource
resource "nsx_service" "foo" {
    name = "foo_service_http_80"
    scopeid = "globalroot-0"
    desc = "foo tcp port 80 service"
    proto = "TCP"
    ports = "80"
}
```

## Argument Reference

The following arguments are used to configure the VMware NSX Provider:

* `nsxusername` - (Required) This is the username for NSX API operations. Can also
  be specified with the `NSX_USER` environment variable.
* `nsxpassword` - (Required) This is the password for NSX API operations. Can
  also be specified with the `NSX_PASSWORD` environment variable.
* `nsxserver` - (Required) This is the NSX server name for NSX API
  operations. Can also be specified with the `NSX_SERVER` environment
  variable.
* `insecure` - (Optional) Boolean that can be set to true to
  disable SSL certificate verification. This should be used with care as it
  could allow an attacker to intercept your auth token. If omitted, default
  value is `false`. Can also be specified with the `NSX_INSECURE`
  environment variable.
* `debug` - (Optional) Boolean to set the gonsx api to log xml calls
   to disk.  The log files are logged to `${HOME}/.govc`, the same path used by
  `govc`.

## Required Privileges

In order to use Terraform provider as non priviledged user, a Role within
vCenter must be assigned the following privileges:

## Acceptance Tests

The VMware NSX provider's acceptance tests require the above provider
configuration fields to be set using the documented environment variables.

Once all these variables are in place, the tests can be run like this:

```
make testacc TEST=./builtin/providers/nsx

```


