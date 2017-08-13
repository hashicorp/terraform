---
layout: "netapp"
page_title: "Provider: NetApp OCCM"
sidebar_current: "docs-netapp-index"
description: |-
  The NetApp cloud provider is used to interact with volume resources supported by the OCCM. The provider needs to be configured with credentials for the OCCM API.
---

# NetApp Provider

The NetApp cloud provider is used to interact with volume resources supported by the OCCM. The provider needs to be configured with credentials for the OCCM API.

Use the navigation to the left to read about the available resources.

## Example Usage

```hcl
# Configure the OCCM API
provider "netapp" {
  host            = "..."
  email           = "..."
  password        = "..."
}

# Create a volume
resource "netapp_cloud_volume" "my-volume" {
  name = "my_nfs_volume"
  type = "nfs"
}
```

## Argument Reference

The following arguments are supported:

* `host` - The hostname to use, OCCM host address. It can also
  be sourced from the `NETAPP_HOST` environment variable.

* `email` - The email associated with the OCCM user. It can also be sourced from
  the `NETAPP_EMAIL` environment variable.

* `password` - The password to use. It can also be sourced from
  the `NETAPP_PASSWORD` environment variable.

## Testing

Credentials must be provided via the `NETAPP_HOST`, `NETAPP_EMAIL`
and `NETAPP_PASSWORD` environment variables in order to run
acceptance tests.
