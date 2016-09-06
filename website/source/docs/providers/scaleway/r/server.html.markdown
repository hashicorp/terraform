---
layout: "scaleway"
page_title: "Scaleway: server"
sidebar_current: "docs-scaleway-resource-server"
description: |-
  Manages Scaleway servers.
---

# scaleway\server

Provides ARM servers. This allows servers to be created, updated and deleted.
For additional details please refer to [API documentation](https://developer.scaleway.com/#servers).

## Example Usage

```
resource "scaleway_server" "test" {
  name = "test"
  image = "5faef9cd-ea9b-4a63-9171-9e26bec03dbc"
  type = "C1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) name of ARM server
* `image` - (Required) base image of ARM server
* `type` - (Required) type of ARM server
* `bootscript` - (Optional) server bootscript, can be bootscript id or name. ex: "x86_64 4.5.7 docker \#4"
* `tags` - (Optional) list of tags for server
* `enable_ipv6` - (Optional) enable ipv6
* `dynamic_ip_required` - (Optional) make server publicly available
* `security_group` - (Optional) assign security group to server
* `volumes` - (Optional) list of sizes (in GB) of extra volumes to create

Field `name`, `type`, `tags`, `dynamic_ip_required`, `security_group` are editable.

## Attributes Reference

The following attributes are exported:

* `id` - id of the new resource
* `private_ip` - private ip of the new resource
* `public_ip` - public ip of the new resource
