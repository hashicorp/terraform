---
layout: "openstack"
page_title: "Provider: OpenStack"
sidebar_current: "docs-openstack-index"
description: |-
  The OpenStack provider is used to interact with the many resources supported by OpenStack. The provider needs to be configured with the proper credentials before it can be used.
---

# OpenStack Provider

The OpenStack provider is used to interact with the
many resources supported by OpenStack. The provider needs to be configured
with the proper credentials before it can be used.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Configure the OpenStack Provider
provider "openstack" {
    username  = "admin"
    tenant_name = "admin"
    password  = "pwd"
    auth_url  = "http://myauthurl:5000/v2.0"
    region    = "RegionOne"
}

# Create a web server
resource "openstack_compute_instance" "test-server" {
    ...
}
```

## Configuration Reference

The following arguments are supported:

* `username` - (Required)

* `tenant_name` - (Required)

* `password` - (Required)

* `auth_url` - (Required)

* `region` - (Required)
