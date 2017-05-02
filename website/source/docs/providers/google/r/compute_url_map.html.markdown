---
layout: "google"
page_title: "Google: google_compute_url_map"
sidebar_current: "docs-google-compute-url-map"
description: |-
  Manages a URL Map resource in GCE.
---

# google\_compute\_url\_map

Manages a URL Map resource within GCE. For more information see
[the official documentation](https://cloud.google.com/compute/docs/load-balancing/http/url-map)
and
[API](https://cloud.google.com/compute/docs/reference/latest/urlMaps).


## Example Usage

```hcl
resource "google_compute_url_map" "foobar" {
  name        = "urlmap"
  description = "a description"

  default_service = "${google_compute_backend_service.home.self_link}"

  host_rule {
    hosts        = ["mysite.com"]
    path_matcher = "allpaths"
  }

  path_matcher {
    name            = "allpaths"
    default_service = "${google_compute_backend_service.home.self_link}"

    path_rule {
      paths   = ["/home"]
      service = "${google_compute_backend_service.home.self_link}"
    }

    path_rule {
      paths   = ["/login"]
      service = "${google_compute_backend_service.login.self_link}"
    }

    path_rule {
      paths   = ["/static"]
      service = "${google_compute_backend_bucket.static.self_link}"
    }
  }

  test {
    service = "${google_compute_backend_service.home.self_link}"
    host    = "hi.com"
    path    = "/home"
  }
}

resource "google_compute_backend_service" "login" {
  name        = "login-backend"
  port_name   = "http"
  protocol    = "HTTP"
  timeout_sec = 10

  health_checks = ["${google_compute_http_health_check.default.self_link}"]
}

resource "google_compute_backend_service" "home" {
  name        = "home-backend"
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

resource "google_compute_backend_bucket" "static" {
  name        = "static-asset-backend-bucket"
  bucket_name = "${google_storage_bucket.static.name}"
  enable_cdn  = true
}

resource "google_storage_bucket" "static" {
  name     = "static-asset-bucket"
  location = "US"
}
```

## Argument Reference

The following arguments are supported:

* `default_service` - (Required) The URL of the backend service or backend bucket to use when none
    of the given rules match. See the documentation for formatting the service/bucket
    URL
    [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#defaultService)

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

- - -

* `description` - (Optional) A brief description of this resource.

* `host_rule` - (Optional) A list of host rules. See below for configuration
    options.

* `path_matcher` - (Optional) A list of paths to match. See below for
    configuration options.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `test` - (Optional) The test to perform. See below for configuration options.

The `host_rule` block supports: (This block can be defined multiple times).

* `hosts` (Required) - A list of hosts to match against. See the documentation
    for formatting each host
    [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#hostRules.hosts)

* `description` - (Optional) An optional description of the host rule.

* `path_matcher` - (Required) The name of the `path_matcher` (defined below)
    to apply this host rule to.

The `path_matcher` block supports: (This block can be defined multiple times)

* `default_service` - (Required) The URL for the backend service or backend bucket to use if none
    of the given paths match. See the documentation for formatting the service/bucket
    URL [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#pathMatcher.defaultService)

* `name` - (Required) The name of the `path_matcher` resource. Used by the
    `host_rule` block above.

* `description` - (Optional) An optional description of the host rule.

The `path_matcher.path_rule` sub-block supports: (This block can be defined
multiple times)

* `paths` - (Required) The list of paths to match against. See the
    documentation for formatting these [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#pathMatchers.pathRules.paths)

* `service` - (Required) The URL for the backend service or backend bucket to use if any
    of the given paths match. See the documentation for formatting the service/bucket
    URL [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#pathMatcher.defaultService)

The optional `test` block supports: (This block can be defined multiple times)

* `service` - (Required) The backend service or backend bucket that should be matched by this test.

* `host` - (Required) The host component of the URL being tested.

* `path` - (Required) The path component of the URL being tested.

* `description` - (Optional) An optional description of this test.

## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `fingerprint` - The unique fingerprint for this resource.

* `id` - The GCE assigned ID of the resource.

* `self_link` - The URI of the created resource.
