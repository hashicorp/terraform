---
layout: "scaleway"
page_title: "Scaleway: ip"
sidebar_current: "docs-scaleway-resource-ip"
description: |-
  Manages Scaleway IPs.
---

# scaleway\ip

Provides IPs for ARM servers. This allows IPs to be created, updated and deleted.
For additional details please refer to [API documentation](https://developer.scaleway.com/#ips).

## Example Usage

```
resource "scaleway_ip" "test_ip" {
}
```

## Argument Reference

The following arguments are supported:

* `server` - (Optional) ID of ARM server to associate IP with

Field `server` is editable.

## Attributes Reference

The following attributes are exported:

* `id` - id of the new resource
* `ip` - IP of the new resource
