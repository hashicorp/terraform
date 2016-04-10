---
layout: "google"
page_title: "Google: google_compute_global_forwarding_rule"
sidebar_current: "docs-google-compute-global-forwarding-rule"
description: |-
  Manages a Target Pool within GCE.
---

# google\_compute\_global\_forwarding\_rule

Manages a Global Forwarding Rule within GCE. This binds an ip and port to a target HTTP(s) proxy. For more
information see [the official
documentation](https://cloud.google.com/compute/docs/load-balancing/network/forwarding-rules) and
[API](https://cloud.google.com/compute/docs/reference/latest/globalForwardingRules).

## Example Usage

```js
resource "google_compute_global_forwarding_rule" "default" {
  name       = "test"
  target     = "${google_compute_target_http_proxy.default.self_link}"
  port_range = "80"
}

resource "google_compute_target_http_proxy" "default" {
  name        = "test-proxy"
  description = "a description"
  url_map     = "${google_compute_url_map.default.self_link}"
}

resource "google_compute_url_map" "default" {
  name            = "url-map"
  description     = "a description"
  default_service = "${google_compute_backend_service.default.self_link}"

  host_rule {
    hosts        = ["mysite.com"]
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = "${google_compute_backend_service.default.self_link}"
    path_rule {
      paths   = ["/*"]
      service = "${google_compute_backend_service.default.self_link}"
    }
  }
}

resource "google_compute_backend_service" "default" {
  name        = "default-backend"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 10

  health_checks = ["${google_compute_http_health_check.default.self_link}"]
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

* `name` - (Required) A unique name for the resource, required by GCE. Changing
  this forces a new resource to be created.

* `target` - (Required) URL of target HTTP or HTTPS proxy.

- - -

* `description` - (Optional) Textual description field.

* `ip_address` - (Optional) The static IP. (if not set, an ephemeral IP is
    used).

* `ip_protocol` - (Optional) The IP protocol to route, one of "TCP" "UDP" "AH"
    "ESP" or "SCTP". (default "TCP").

* `port_range` - (Optional) A range e.g. "1024-2048" or a single port "1024"
    (defaults to all ports!).

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.
