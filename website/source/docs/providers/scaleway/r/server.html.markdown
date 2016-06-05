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

Field `name`, `type` are editable.

## Attributes Reference

The following attributes are exported:

* `id` - id of the new resource
