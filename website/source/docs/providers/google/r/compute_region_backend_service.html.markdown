---
layout: "google"
page_title: "Google: google_compute_region_backend_service"
sidebar_current: "docs-google-compute-region-backend-service"
description: |-
  Creates a Region Backend Service resource for Google Compute Engine.
---

# google\_compute\_region\_backend\_service

A Region Backend Service defines a regionally-scoped group of virtual machines that will serve traffic for load balancing.

See [backendServices](https://cloud.google.com/compute/docs/reference/latest/backendServices) documentation for more on this resource type, and [Internal Load Balancing](https://cloud.google.com/compute/docs/load-balancing/internal/) documentation for more details on usage.

## Example Usage

```tf
resource "google_compute_region_backend_service" "foobar" {
  name                  = "blablah"
  description           = "Hello World 1234"
  protocol              = "TCP"
  timeout_sec           = 10

  backend {
    group = "${google_compute_instance_group_manager.foo.instance_group}"
  }

  health_checks = ["${google_compute_health_check.default.self_link}"]
}

resource "google_compute_instance_group_manager" "foo" {
  name               = "terraform-test"
  instance_template  = "${google_compute_instance_template.foobar.self_link}"
  base_instance_name = "foobar"
  zone               = "us-central1-f"
  target_size        = 1
}

resource "google_compute_instance_template" "foobar" {
  name         = "terraform-test"
  machine_type = "n1-standard-1"

  network_interface {
    network = "default"
  }

  disk {
    source_image = "debian-cloud/debian-8"
    auto_delete  = true
    boot         = true
  }
}

resource "google_compute_health_check" "default" {
  name               = "test"
  check_interval_sec = 1
  timeout_sec        = 1
  type               = "TCP"
  tcp_health_check {
    port = "80"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the backend service.

* `health_checks` - (Required) Specifies a list of health check objects
    for checking the health of the backend service.

- - -

* `backend` - (Optional) The list of backends that serve this BackendService.
    See *Backend* below.

* `description` - (Optional) The textual description for the backend service.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `protocol` - (Optional) The protocol for incoming requests. Defaults to
    `HTTP`.

* `region` - (Optional) The Region in which the created address should reside.
    If it is not provided, the provider region is used.

* `timeout_sec` - (Optional) The number of secs to wait for a backend to respond
    to a request before considering the request failed. Defaults to `30`.


**Backend** supports the following attributes:

* `group` - (Required) The name or URI of a Compute Engine instance group
    (`google_compute_instance_group_manager.xyz.instance_group`) that can
    receive traffic.

* `balancing_mode` - (Optional) Defines the strategy for balancing load.
    Defaults to `UTILIZATION`

* `capacity_scaler` - (Optional) A float in the range [0, 1.0] that scales the
    maximum parameters for the group (e.g., max rate). A value of 0.0 will cause
    no requests to be sent to the group (i.e., it adds the group in a drained
    state). The default is 1.0.

* `description` - (Optional) Textual description for the backend.

* `max_rate` - (Optional) Maximum requests per second (RPS) that the group can
    handle.

* `max_rate_per_instance` - (Optional) The maximum per-instance requests per
    second (RPS).

* `max_utilization` - (Optional) The target CPU utilization for the group as a
    float in the range [0.0, 1.0]. This flag can only be provided when the
    balancing mode is `UTILIZATION`. Defaults to `0.8`.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `fingerprint` - The fingerprint of the backend service.

* `self_link` - The URI of the created resource.
