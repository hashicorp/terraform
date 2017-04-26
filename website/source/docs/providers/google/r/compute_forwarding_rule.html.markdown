---
layout: "google"
page_title: "Google: google_compute_forwarding_rule"
sidebar_current: "docs-google-compute-forwarding-rule"
description: |-
  Manages a Forwarding Rule within GCE.
---

# google\_compute\_forwarding\_rule

Manages a Forwarding Rule within GCE. This binds an ip and port range to a target pool. For more
information see [the official
documentation](https://cloud.google.com/compute/docs/load-balancing/network/forwarding-rules) and
[API](https://cloud.google.com/compute/docs/reference/latest/forwardingRules).

## Example Usage

```tf
resource "google_compute_forwarding_rule" "default" {
  name       = "test"
  target     = "${google_compute_target_pool.default.self_link}"
  port_range = "80"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE. Changing
    this forces a new resource to be created.

- - -

* `backend_service` - (Optional) BackendService resource to receive the
    matched traffic. Only used for internal load balancing.

* `description` - (Optional) Textual description field.

* `ip_address` - (Optional) The static IP. (if not set, an ephemeral IP is
    used).

* `ip_protocol` - (Optional) The IP protocol to route, one of "TCP" "UDP" "AH"
    "ESP" or "SCTP" for external load balancing, "TCP" or "UDP" for internal
    (default "TCP").

* `load_balancing_scheme` - (Optional) Type of load balancing to use. Can be
    set to "INTERNAL" or "EXTERNAL" (default "EXTERNAL").

* `network` - (Optional) Network that the load balanced IP should belong to.
    Only used for internal load balancing. If it is not provided, the default
    network is used.

* `port_range` - (Optional) A range e.g. "1024-2048" or a single port "1024"
    (defaults to all ports!). Only used for external load balancing.

* `ports` - (Optional) A list of ports (maximum of 5) to use for internal load
    balancing. Packets addressed to these ports will be forwarded to the backends
    configured with this forwarding rule. Required for internal load balancing.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `region` - (Optional) The Region in which the created address should reside.
    If it is not provided, the provider region is used.

* `subnetwork` - (Optional) Subnetwork that the load balanced IP should belong
    to. Only used for internal load balancing. Must be specified if the network
    is in custom subnet mode.

* `target` - (Optional) URL of target pool. Required for external load
    balancing.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.
