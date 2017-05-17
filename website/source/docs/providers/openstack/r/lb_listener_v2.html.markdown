---
layout: "openstack"
page_title: "OpenStack: openstack_lb_listener_v2"
sidebar_current: "docs-openstack-resource-lb-listener-v2"
description: |-
  Manages a V2 listener resource within OpenStack.
---

# openstack\_lb\_listener\_v2

Manages a V2 listener resource within OpenStack.

## Example Usage

```hcl
resource "openstack_lb_listener_v2" "listener_1" {
  protocol        = "HTTP"
  protocol_port   = 8080
  loadbalancer_id = "d9415786-5f1a-428b-b35f-2f1523e146d2"
}
```

## Argument Reference

The following arguments are supported:

* `region` - (Required) The region in which to obtain the V2 Networking client.
    A Networking client is needed to create an . If omitted, the
    `OS_REGION_NAME` environment variable is used. Changing this creates a new
    Listener.

* `protocol` = (Required) The protocol - can either be TCP, HTTP or HTTPS.
    Changing this creates a new Listener.

* `protocol_port` = (Required) The port on which to listen for client traffic.
    Changing this creates a new Listener.

* `tenant_id` - (Optional) Required for admins. The UUID of the tenant who owns
    the Listener.  Only administrative users can specify a tenant UUID
    other than their own. Changing this creates a new Listener.

* `loadbalancer_id` - (Required) The load balancer on which to provision this
    Listener. Changing this creates a new Listener.

* `name` - (Optional) Human-readable name for the Listener. Does not have
    to be unique.

* `default_pool_id` - (Optional) The ID of the default pool with which the
    Listener is associated. Changing this creates a new Listener.

* `description` - (Optional) Human-readable description for the Listener.

* `connection_limit` - (Optional) The maximum number of connections allowed
    for the Listener.

* `default_tls_container_ref` - (Optional) A reference to a container of TLS
    secrets.

* `sni_container_refs` - (Optional) A list of references to TLS secrets.

* `admin_state_up` - (Optional) The administrative state of the Listener.
    A valid value is true (UP) or false (DOWN).

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID for the Listener.
* `protocol` - See Argument Reference above.
* `protocol_port` - See Argument Reference above.
* `tenant_id` - See Argument Reference above.
* `name` - See Argument Reference above.
* `default_port_id` - See Argument Reference above.
* `description` - See Argument Reference above.
* `connection_limit` - See Argument Reference above.
* `default_tls_container_ref` - See Argument Reference above.
* `sni_container_refs` - See Argument Reference above.
* `admin_state_up` - See Argument Reference above.
