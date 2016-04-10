---
layout: "google"
page_title: "Google: google_compute_target_pool"
sidebar_current: "docs-google-compute-target-pool"
description: |-
  Manages a Target Pool within GCE.
---

# google\_compute\_target\_pool

Manages a Target Pool within GCE. This is a collection of instances used as
target of a network load balancer (Forwarding Rule). For more information see
[the official
documentation](https://cloud.google.com/compute/docs/load-balancing/network/target-pools)
and [API](https://cloud.google.com/compute/docs/reference/latest/targetPools).


## Example Usage

```js
resource "google_compute_target_pool" "default" {
  name = "test"

  instances = [
    "us-central1-a/myinstance1",
    "us-central1-b/myinstance2",
  ]

  health_checks = [
    "${google_compute_http_health_check.default.name}",
  ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE. Changing
    this forces a new resource to be created.

- - -

* `backup_pool` - (Optional) URL to the backup target pool. Must also set
    failover\_ratio.

* `description` - (Optional) Textual description field.

* `failover_ratio` - (Optional) Ratio (0 to 1) of failed nodes before using the
    backup pool (which must also be set).

* `health_checks` - (Optional) List of zero or one healthcheck names.

* `instances` - (Optional) List of instances in the pool. They can be given as
    URLs, or in the form of "zone/name". Note that the instances need not exist
    at the time of target pool creation, so there is no need to use the
    Terraform interpolators to create a dependency on the instances from the
    target pool.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `region` - (Optional) Where the target pool resides. Defaults to project
    region.

* `session_affinity` - (Optional) How to distribute load. Options are "NONE" (no
    affinity). "CLIENT\_IP" (hash of the source/dest addresses / ports), and
    "CLIENT\_IP\_PROTO" also includes the protocol (default "NONE").

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.
