---
layout: "clc"
page_title: "clc: clc_load_balancer"
sidebar_current: "docs-clc-resource-load-balancer"
description: |-
  Manages a CLC load balacner.
---

# clc_load_balancer

Manages a CLC load balancer. Manage connected backends with [clc_load_balancer_pool](load_balancer_pool.html)

See also [Complete API documentation](https://www.ctl.io/api-docs/v2/#shared-load-balancer).

## Example Usage

```hcl
# Provision a load balancer
resource "clc_load_balancer" "api" {
  data_center = "${clc_group.frontends.location_id}"
  name        = "api"
  description = "api load balancer"
  status      = "enabled"
}

output "api_ip" {
  value = "clc_load_balancer.api.ip_address"
}
```


## Argument Reference

The following arguments are supported:

* `name` - (Required, string) The name of the load balancer.
* `data_center` - (Required, string) The datacenter location of both parent group and this group.
* `status` - (Required, string) Either "enabled" or "disabled"
* `description` - (Optional, string) Description for server group (visible in control portal only)
