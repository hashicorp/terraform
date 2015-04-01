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
    user_name  = "admin"
    tenant_name = "admin"
    password  = "pwd"
    auth_url  = "http://myauthurl:5000/v2.0"
}

# Create a web server
resource "openstack_compute_instance_v2" "test-server" {
    ...
}
```

## Configuration Reference

The following arguments are supported:

* `auth_url` - (Required)

* `user_name` - (Optional; Required for Identity V2)

* `user_id` - (Optional)

* `password` - (Optional; Required if not using `api_key`)

* `api_key` - (Optional; Required if not using `password`)

* `domain_id` - (Optional)

* `domain_name` - (Optional)

* `tenant_id` - (Optional)

* `tenant_name` - (Optional)
