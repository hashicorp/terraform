---
layout: "google"
page_title: "Google: google_compute_url_map"
sidebar_current: "docs-google-resource-url-map"
description: |-
  Manages a URL Map resource in GCE.
---

# google\_compute\_url\_map

Manages a URL Map resource within GCE.  For more information see
[the official documentation](https://cloud.google.com/compute/docs/load-balancing/http/url-map)
and
[API](https://cloud.google.com/compute/docs/reference/latest/urlMaps).


## Example Usage

```
resource "google_compute_url_map" "foobar" {
    name = "urlmap"
    description = "a description"
    default_service = "${google_compute_backend_service.home.self_link}"

    host_rule {
        hosts = ["mysite.com"]
        path_matcher = "allpaths"
    }

    path_matcher {
        default_service = "${google_compute_backend_service.home.self_link}"
        name = "allpaths"
        path_rule {
            paths = ["/home"]
            service = "${google_compute_backend_service.home.self_link}"
        }

        path_rule {
            paths = ["/login"]
            service = "${google_compute_backend_service.login.self_link}"
        }
    }

    test {
        service = "${google_compute_backend_service.home.self_link}"
        host = "hi.com"
        path = "/home"
    }
}

resource "google_compute_backend_service" "login" {
    name = "login-backend"
    port_name = "http"
    protocol = "HTTP"
    timeout_sec = 10
    region = "us-central1"

    health_checks = ["${google_compute_http_health_check.default.self_link}"]
}

resource "google_compute_backend_service" "home" {
    name = "home-backend"
    port_name = "http"
    protocol = "HTTP"
    timeout_sec = 10
    region = "us-central1"

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

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

* `description` - (Optional) A brief description of this resource.

* `default_service` - (Required) The URL of the backend service to use when none of the
    given rules match. See the documentation for formatting the service URL 
    [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#defaultService)

The `host_rule` block supports: (Note that this block can be defined an arbitrary
number of times.)

* `hosts` (Required) - A list of hosts to match against. See the documention
    for formatting each host [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#hostRules.hosts)

* `description` - (Optional) An optional description of the host rule.

* `path_matcher` - (Required) The name of the `path_matcher` (defined below) 
    to apply this host rule to. 

The `path_matcher` block supports: (Note that this block can be defined an arbitrary
number of times.)

* `default_service` - (Required) The URL for the backend service to use if none
    of the given paths match. See the documentation for formatting the service URL 
    [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#pathMatcher.defaultService)

* `name` - (Required) The name of the `path_matcher` resource. Used by the `host_rule`
    block above.

* `description` - (Optional) An optional description of the host rule.

The `path_matcher.path_rule` sub-block supports: (Note that this block can be defined an arbitrary
number of times.)

* `paths` - (Required) The list of paths to match against. See the
    documentation for formatting these [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#pathMatchers.pathRules.paths)

* `default_service` - (Required) The URL for the backend service to use if any
    of the given paths match. See the documentation for formatting the service URL 
    [here](https://cloud.google.com/compute/docs/reference/latest/urlMaps#pathMatcher.defaultService)

The optional `test` block supports: (Note that this block can be defined an arbitary 
number of times.)

* `service` - (Required) The service that should be matched by this test.

* `host` - (Required) The host component of the URL being tested.

* `path` - (Required) The path component of the URL being tested.

* `description` - (Optional) An optional description of this test.

## Attributes Reference

The following attributes are exported:

* `id` - The GCE assigned ID of the resource.
* `self_link` - A GCE assigned link to the resource.
