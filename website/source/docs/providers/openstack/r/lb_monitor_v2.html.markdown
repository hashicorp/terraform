---
layout: "openstack"
page_title: "OpenStack: openstack_lb_monitor_v2"
sidebar_current: "docs-openstack-resource-lb-monitor-v2"
description: |-
  Manages a V2 monitor resource within OpenStack.
---

# openstack\_lb\_monitor\_v2

Manages a V2 monitor resource within OpenStack.

## Example Usage

```hcl
resource "openstack_lb_monitor_v2" "monitor_1" {
  pool_id     = "${openstack_lb_pool_v2.pool_1.id}"
  type        = "PING"
  delay       = 20
  timeout     = 10
  max_retries = 5
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create an . If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    monitor.
    
* `pool_id` - (Required) The id of the pool that this monitor will be assigned to.

* `name` - (Optional) The Name of the Monitor.

* `tenant_id` - (Optional) Required for admins. The UUID of the tenant who owns
    the monitor.  Only administrative users can specify a tenant UUID
    other than their own. Changing this creates a new monitor.

* `type` - (Required) The type of probe, which is PING, TCP, HTTP, or HTTPS,
    that is sent by the load balancer to verify the member state. Changing this
    creates a new monitor.

* `delay` - (Required) The time, in seconds, between sending probes to members.

* `timeout` - (Required) Maximum number of seconds for a monitor to wait for a
    ping reply before it times out. The value must be less than the delay
    value.

* `max_retries` - (Required) Number of permissible ping failures before
    changing the member's status to INACTIVE. Must be a number between 1
    and 10..

* `url_path` - (Optional) Required for HTTP(S) types. URI path that will be
    accessed if monitor type is HTTP or HTTPS.

*  `http_method` - (Optional) Required for HTTP(S) types. The HTTP method used
    for requests by the monitor. If this attribute is not specified, it
    defaults to "GET".

* `expected_codes` - (Optional) Required for HTTP(S) types. Expected HTTP codes
    for a passing HTTP(S) monitor. You can either specify a single status like
    "200", or a range like "200-202".

* `admin_state_up` - (Optional) The administrative state of the monitor.
    A valid value is true (UP) or false (DOWN).


## Attributes Reference

The following attributes are exported:

* `id` - The unique ID for the monitor.
* `tenant_id` - See Argument Reference above.
* `type` - See Argument Reference above.
* `delay` - See Argument Reference above.
* `timeout` - See Argument Reference above.
* `max_retries` - See Argument Reference above.
* `url_path` - See Argument Reference above.
* `http_method` - See Argument Reference above.
* `expected_codes` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
