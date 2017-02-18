---
layout: "google"
page_title: "Google: google_compute_health_check"
sidebar_current: "docs-google-compute-health-check"
description: |-
  Manages a Health Check within GCE.
---

# google\_compute\_health\_check

Manages a health check within GCE. This is used to monitor instances
behind load balancers. Timeouts or HTTP errors cause the instance to be
removed from the pool. For more information, see [the official
documentation](https://cloud.google.com/compute/docs/load-balancing/health-checks)
and
[API](https://cloud.google.com/compute/docs/reference/latest/healthChecks).

## Example Usage

```tf
resource "google_compute_health_check" "default" {
  name = "test"

  timeout_sec        = 1
  check_interval_sec = 1

  tcp_health_check {
    port = "80"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) A unique name for the resource, required by GCE.
    Changing this forces a new resource to be created.

- - -

* `check_interval_sec` - (Optional) The number of seconds between each poll of
    the instance instance (default 5).

* `description` - (Optional) Textual description field.

* `healthy_threshold` - (Optional) Consecutive successes required (default 2).

* `http_health_check` - (Optional) An HTTP Health Check.
    See *HTTP Health Check* below.

* `https_health_check` - (Optional) An HTTPS Health Check.
    See *HTTPS Health Check* below.

* `ssl_health_check` - (Optional) An SSL Health Check.
    See *SSL Health Check* below.

* `tcp_health_check` - (Optional) A TCP Health Check.
    See *TCP Health Check* below.

* `project` - (Optional) The project in which the resource belongs. If it
    is not provided, the provider project is used.

* `timeout_sec` - (Optional) The number of seconds to wait before declaring
    failure (default 5).

* `unhealthy_threshold` - (Optional) Consecutive failures required (default 2).


**HTTP Health Check** supports the following attributes:

* `host` - (Optional) HTTP host header field (default instance's public ip).

* `port` - (Optional) TCP port to connect to (default 80).

* `proxy_header` - (Optional) Type of proxy header to append before sending
    data to the backend, either NONE or PROXY_V1 (default NONE).

* `request_path` - (Optional) URL path to query (default /).


**HTTPS Health Check** supports the following attributes:

* `host` - (Optional) HTTPS host header field (default instance's public ip).

* `port` - (Optional) TCP port to connect to (default 443).

* `proxy_header` - (Optional) Type of proxy header to append before sending
    data to the backend, either NONE or PROXY_V1 (default NONE).

* `request_path` - (Optional) URL path to query (default /).


**SSL Health Check** supports the following attributes:

* `port` - (Optional) TCP port to connect to (default 443).

* `proxy_header` - (Optional) Type of proxy header to append before sending
    data to the backend, either NONE or PROXY_V1 (default NONE).

* `request` - (Optional) Application data to send once the SSL connection has
    been established (default "").

* `response` - (Optional) The response that indicates health (default "")


**TCP Health Check** supports the following attributes:

* `port` - (Optional) TCP port to connect to (default 80).

* `proxy_header` - (Optional) Type of proxy header to append before sending
    data to the backend, either NONE or PROXY_V1 (default NONE).

* `request` - (Optional) Application data to send once the TCP connection has
    been established (default "").

* `response` - (Optional) The response that indicates health (default "")


## Attributes Reference

In addition to the arguments listed above, the following computed attributes are
exported:

* `self_link` - The URI of the created resource.
