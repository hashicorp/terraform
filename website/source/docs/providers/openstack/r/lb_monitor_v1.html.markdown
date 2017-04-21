---
layout: "openstack"
page_title: "OpenStack: openstack_lb_monitor_v1"
sidebar_current: "docs-openstack-resource-lb-monitor-v1"
description: |-
  Manages a V1 load balancer monitor resource within OpenStack.
---

# openstack\_lb\_monitor_v1

Manages a V1 load balancer monitor resource within OpenStack.

## Example Usage

```hcl
resource "openstack_lb_monitor_v1" "monitor_1" {
  type           = "PING"
  delay          = 30
  timeout        = 5
  max_retries    = 3
  admin_state_up = "true"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create an LB monitor. If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    LB monitor.

* `type` - (Required) The type of probe, which is PING, TCP, HTTP, or HTTPS,
    that is sent by the monitor to verify the member state. Changing this
    creates a new monitor.

* `delay` - (Required) The time, in seconds, between sending probes to members.
    Changing this creates a new monitor.

* `timeout` - (Required) Maximum number of seconds for a monitor to wait for a
    ping reply before it times out. The value must be less than the delay value.
    Changing this updates the timeout of the existing monitor.

* `max_retries` - (Required) Number of permissible ping failures before changing
    the member's status to INACTIVE. Must be a number between 1 and 10. Changing
    this updates the max_retries of the existing monitor.

* `url_path` - (Optional) Required for HTTP(S) types. URI path that will be
    accessed if monitor type is HTTP or HTTPS. Changing this updates the
    url_path of the existing monitor.

* `http_method` - (Optional) Required for HTTP(S) types. The HTTP method used
    for requests by the monitor. If this attribute is not specified, it defaults
    to "GET". Changing this updates the http_method of the existing monitor.

* `expected_codes` - (Optional) equired for HTTP(S) types. Expected HTTP codes
    for a passing HTTP(S) monitor. You can either specify a single status like
    "200", or a range like "200-202". Changing this updates the expected_codes
    of the existing monitor.

* `admin_state_up` - (Optional) The administrative state of the monitor.
    Acceptable values are "true" and "false". Changing this value updates the
    state of the existing monitor.

* `tenant_id` - (Optional) The owner of the monitor. Required if admin wants to
    create a monitor for another tenant. Changing this creates a new monitor.

## Attributes Reference

The following attributes are exported:

* `region` - See Argument Reference above.
* `type` - See Argument Reference above.
* `delay` - See Argument Reference above.
* `timeout` - See Argument Reference above.
* `max_retries` - See Argument Reference above.
* `url_path` - See Argument Reference above.
* `http_method` - See Argument Reference above.
* `expected_codes` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.

## Import

Load Balancer Members can be imported using the `id`, e.g.

```
$ terraform import openstack_lb_monitor_v1.monitor_1 119d7530-72e9-449a-aa97-124a5ef1992c
```
