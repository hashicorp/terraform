---
layout: "google"
page_title: "Google: google_compute_target_https_proxy"
sidebar_current: "docs-google-compute-target-https-proxy"
description: |-
  Creates a Target HTTPS Proxy resource in GCE.
---

# google\_compute\_target\_https\_proxy

Creates a target HTTPS proxy resource in GCE. For more information see
[the official
documentation](https://cloud.google.com/compute/docs/load-balancing/http/target-proxies) and
[API](https://cloud.google.com/compute/docs/reference/latest/targetHttpsProxies).


## Example Usage

```hcl
resource "google_compute_target_https_proxy" "default" {
  name             = "test-proxy"
  description      = "a description"
  url_map          = "${google_compute_url_map.default.self_link}"
  ssl_certificates = ["${google_compute_ssl_certificate.default.self_link}"]
}

resource "google_compute_ssl_certificate" "default" {
  name        = "my-certificate"
  description = "a description"
  private_key = "${file("path/to/private.key")}"
  certificate = "${file("path/to/certificate.crt")}"
}

resource "google_compute_url_map" "default" {
  name        = "url-map"
  description = "a description"

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

* `ssl_certificates` - (Required) The URLs of the SSL Certificate resources that
    authenticate connections between users and load balancing. Currently exactly
    one must be specified.

* `url_map` - (Required) The URL of a URL Map resource that defines the mapping
    from the URL to the BackendService.

- - -

* `description` - (Optional) A description of this resource. Changing this
    forces a new resource to be created.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `id` - A unique ID assigned by GCE.

* `self_link` - The URI of the created resource.
