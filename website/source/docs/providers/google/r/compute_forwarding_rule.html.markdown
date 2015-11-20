---
layout: "google"
page_title: "Google: google_compute_forwarding_rule"
sidebar_current: "docs-google-compute-forwarding-rule"
description: |-
  Manages a Target Pool within GCE.
---

# google\_compute\_forwarding\_rule

Manages a Forwarding Rule within GCE.  This binds an ip and port range to a target pool.  For more
information see [the official
documentation](https://cloud.google.com/compute/docs/load-balancing/network/forwarding-rules) and
[API](https://cloud.google.com/compute/docs/reference/latest/forwardingRules).

## Example Usage

```
resource "google_compute_forwarding_rule" "default" {
	name = "test"
	target = "${google_compute_target_pool.default.self_link}"
	port_range = "80"
}
```

## Argument Reference

The following arguments are supported:

* `description` - (Optional) Textual description field.

* `ip_address` - (Optional) The static IP. (if not set, an ephemeral IP is
used).

* `ip_protocol` - (Optional) The IP protocol to route, one of "TCP" "UDP" "AH" "ESP" or "SCTP". (default "TCP").

* `name` - (Required) A unique name for the resource, required by GCE.  Changing
  this forces a new resource to be created.

* `port_range` - (Optional) A range e.g. "1024-2048" or a single port "1024"
(defaults to all ports!).

* `target` - URL of target pool.

## Attributes Reference

The following attributes are exported:

* `self_link` - The URL of the created resource.

* `ip_address` - The IP address that was chosen (or specified).


