---
layout: "clc"
page_title: "clc: clc_load_balancer_pool"
sidebar_current: "docs-clc-resource-load-balancer-pool"
description: |-
  Manages a CLC load balancer pool.
---

# clc\_load\_balancer\_pool

Manages a CLC load balancer pool. Manage related frontend with [clc_load_balancer](load_balancer.html)

See also [Complete API documentation](https://www.ctl.io/api-docs/v2/#shared-load-balancer).

## Example Usage


```
# Provision a load balancer pool
resource "clc_load_balancer_pool" "pool" {
  data_center = "${clc_group.frontends.location_id}"
  load_balancer = "${clc_load_balancer.api.id}"
  method = "roundRobin"
  persistence = "standard"
  port = 80
  nodes
    {
      status = "enabled"
      ipAddress = "${clc_server.node.0.private_ip_address}"
      privatePort = 3000
    }
  nodes
    {
      status = "enabled"
      ipAddress = "${clc_server.node.1.private_ip_address}"
      privatePort = 3000
    }
}

output "pool" {
  value = "$join(" ", clc_load_balancer.pool.nodes)}"
}
```


## Argument Reference

The following arguments are supported:

* `load_balancer` - (Required, string) The id of the load balancer.
* `data_center` - (Required, string) The datacenter location for this pool.
* `port` - (Required, int) Either 80 or 443
* `method` - (Optional, string) The configured balancing method. Either
  "roundRobin" (default) or "leastConnection".
* `persistence` - (Optional, string) The configured persistence
  method. Either "standard" (default) or "sticky".
* nodes - (Optional) See [Nodes](#nodes) below for details. 


<a id="nodes"></a>
## Nodes


`nodes` is a block within the configuration that may be repeated to
specify connected nodes on this pool. Each `nodes` block supports the
following:

* `ipAddress` (Required, string) The destination internal ip of pool node. 
* `privatePort` (Required, int) The destination port on the pool node. 
* `status` (Optional, string) Either "enabled" or "disabled".






