---
layout: "google"
page_title: "Google: google_compute_backend_service"
sidebar_current: "docs-google-resource-backend-service"
description: |-
  Creates a Backend Service resource for Google Compute Engine.
---

# google\_compute\_backend\_service




## Example Usage

```
resource "google_compute_backend_service" "foobar" {
    name = "blablah"
    health_checks = ["${google_compute_http_health_check.default.self_link}"]
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
* `backends` - (Optional) The list of backends that serve this BackendService. See *Backend* below.
* `port_name` - (Optional) The name of a service that has been added to
	an instance group in this backend. See [related docs](https://cloud.google.com/compute/docs/instance-groups/#specifying_service_endpoints)
    for details. Defaults to http.
* `protocol` - (Optional) The protocol for incoming requests. Defaults to `HTTP`.
* `timeout_sec` - (Optional) The number of secs to wait for a backend to respond
	to a request before considering the request failed. Defaults to `30`.

**Backend** supports the following attributes:

* `group` - (Required) The name or URI of a Compute Engine instance group that can receive traffic.
* `balancing_mode` - (Optional) Defines the strategy for balancing load. Defaults to `UTILIZATION`
* `capacity_scaler` - (Optional) A float in the range [0, 1.0] that scales the maximum parameters for the group (e.g., max rate). A value of 0.0 will cause no requests to be sent to the group (i.e., it adds the group in a drained state). The default is 1.0.
* `description` - (Optional) Textual description for the backend.
* `max_rate` - (Optional) Maximum requests per second (RPS) that the group can handle.
* `max_rate_per_instance` - (Optional) The maximum per-instance requests per second (RPS).
* `max_utilization` - (Optional) The target CPU utilization for the group as a float in the range [0.0, 1.0]. This flag can only be provided when the balancing mode is `UTILIZATION`. Defaults to `0.8`.

## Attributes Reference

The following attributes are exported:

* `name` - The name of the resource.
* `backend_service` - The IP backend_service that was allocated.
* `self_link` - The URI of the created resource.
