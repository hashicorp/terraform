---
layout: "google"
page_title: "Google: google_compute_backend_service"
sidebar_current: "docs-google-compute-backend-service"
description: |-
  Creates a Backend Service resource for Google Compute Engine.
---

# google\_compute\_backend\_service

A Backend Service defines a group of virtual machines that will serve traffic for load balancing.

## Example Usage

```
resource "google_compute_backend_service" "foobar" {
    name = "blablah"
    description = "Hello World 1234"
    port_name = "http"
    protocol = "HTTP"
    timeout_sec = 10
    region = "us-central1"

    backend {
        group = "${google_compute_instance_group_manager.foo.instance_group}"
    }

    health_checks = ["${google_compute_http_health_check.default.self_link}"]
}

resource "google_compute_instance_group_manager" "foo" {
    name = "terraform-test"
    instance_template = "${google_compute_instance_template.foobar.self_link}"
    base_instance_name = "foobar"
    zone = "us-central1-f"
    target_size = 1
}

resource "google_compute_instance_template" "foobar" {
    name = "terraform-test"
    machine_type = "n1-standard-1"

    network_interface {
        network = "default"
    }

    disk {
        source_image = "debian-7-wheezy-v20140814"
        auto_delete = true
        boot = true
    }
}

resource "google_compute_http_health_check" "default" {
    name = "test"
    request_path = "/"
    check_interval_sec = 1
    timeout_sec = 1
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the backend service.
* `health_checks` - (Required) Specifies a list of HTTP health check objects
    for checking the health of the backend service.
* `description` - (Optional) The textual description for the backend service.
* `backend` - (Optional) The list of backends that serve this BackendService. See *Backend* below.
* `region` - (Optional) The region the service sits in. If not specified, the project region is used.
* `port_name` - (Optional) The name of a service that has been added to
	an instance group in this backend. See [related docs](https://cloud.google.com/compute/docs/instance-groups/#specifying_service_endpoints)
    for details. Defaults to http.
* `protocol` - (Optional) The protocol for incoming requests. Defaults to `HTTP`.
* `timeout_sec` - (Optional) The number of secs to wait for a backend to respond
	to a request before considering the request failed. Defaults to `30`.

**Backend** supports the following attributes:

* `group` - (Required) The name or URI of a Compute Engine instance group (`google_compute_instance_group_manager.xyz.instance_group`) that can receive traffic.
* `balancing_mode` - (Optional) Defines the strategy for balancing load. Defaults to `UTILIZATION`
* `capacity_scaler` - (Optional) A float in the range [0, 1.0] that scales the maximum parameters for the group (e.g., max rate). A value of 0.0 will cause no requests to be sent to the group (i.e., it adds the group in a drained state). The default is 1.0.
* `description` - (Optional) Textual description for the backend.
* `max_rate` - (Optional) Maximum requests per second (RPS) that the group can handle.
* `max_rate_per_instance` - (Optional) The maximum per-instance requests per second (RPS).
* `max_utilization` - (Optional) The target CPU utilization for the group as a float in the range [0.0, 1.0]. This flag can only be provided when the balancing mode is `UTILIZATION`. Defaults to `0.8`.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `self_link` - The URI of the created resource.
