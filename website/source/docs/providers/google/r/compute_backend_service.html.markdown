---
layout: "google"
page_title: "Google: google_compute_backend_service"
sidebar_current: "docs-google-compute-backend-service"
description: |-
  Creates a Backend Service resource for Google Compute Engine.
---

# google\_compute\_backend\_service

A Backend Service defines a group of virtual machines that will serve traffic for load balancing. For more information 
see [the official documentation](https://cloud.google.com/compute/docs/load-balancing/http/backend-service)
and the [API](https://cloud.google.com/compute/docs/reference/latest/backendServices).

For internal load balancing, use a [google_compute_region_backend_service](/docs/providers/google/r/compute_region_backend_service.html).

## Example Usage

```hcl
resource "google_compute_backend_service" "foobar" {
  name        = "blablah"
  description = "Hello World 1234"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 10
  enable_cdn  = false

  backend {
    group = "${google_compute_instance_group_manager.foo.instance_group}"
  }

  health_checks = ["${google_compute_http_health_check.default.self_link}"]
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

resource "google_compute_http_health_check" "default" {
  name               = "test"
  request_path       = "/"
  check_interval_sec = 1
  timeout_sec        = 1
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the backend service.

* `health_checks` - (Required) Specifies a list of HTTP health check objects
    for checking the health of the backend service.

- - -

* `backend` - (Optional) The list of backends that serve this BackendService. Structure is documented below.

* `description` - (Optional) The textual description for the backend service.

* `enable_cdn` - (Optional) Whether or not to enable the Cloud CDN on the backend service.

* `port_name` - (Optional) The name of a service that has been added to an
    instance group in this backend. See [related docs](https://cloud.google.com/compute/docs/instance-groups/#specifying_service_endpoints) for details. Defaults to http.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `protocol` - (Optional) The protocol for incoming requests. Defaults to
    `HTTP`.

* `session_affinity` - (Optional) How to distribute load. Options are `NONE` (no
    affinity), `CLIENT_IP` (hash of the source/dest addresses / ports), and
    `GENERATED_COOKIE` (distribute load using a generated session cookie).

* `timeout_sec` - (Optional) The number of secs to wait for a backend to respond
    to a request before considering the request failed. Defaults to `30`.
    
* `connection_draining_timeout_sec` - (Optional) Time for which instance will be drained (not accept new connections, 
but still work to finish started ones). Defaults to `0`.

The `backend` block supports:

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
