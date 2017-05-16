---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_loadbalancer"
sidebar_current: "docs-do-resource-loadbalancer"
description: |-
  Provides a DigitalOcean Load Balancer resource. This can be used to create, modify, and delete Load Balancers.
---

# digitalocean\_loadbalancer

Provides a DigitalOcean Load Balancer resource. This can be used to create,
modify, and delete Load Balancers.

## Example Usage

```hcl
resource "digitalocean_droplet" "web" {
  name      = "web-1"
  size      = "512mb"
  image     = "centos-7-x64"
  region    = "nyc3"
}

resource "digitalocean_loadbalancer" "public" {
  name = "loadbalancer-1"
  region = "nyc3"

  forwarding_rule {
    entry_port = 80
    entry_protocol = "http"

    target_port = 80
    target_protocol = "http"
  }

  healthcheck {
    port = 22
    protocol = "tcp"
  }

  droplet_ids = ["${digitalocean_droplet.web.id}"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The Load Balancer name
* `region` - (Required) The region to start in
* `algorithm` - (Optional) The load balancing algorithm used to determine
which backend Droplet will be selected by a client. It must be either `round_robin`
or `least_connections`. The default value is `round_robin`.
* `forwarding_rule` - (Required) A list of `forwarding_rule` to be assigned to the
Load Balancer. The `forwarding_rule` block is documented below.
* `healthcheck` - (Optional) A `healthcheck` block to be assigned to the
Load Balancer. The `healthcheck` block is documented below. Only 1 healthcheck is allowed.
* `sticky_sessions` - (Optional) A `sticky_sessions` block to be assigned to the
Load Balancer. The `sticky_sessions` block is documented below. Only 1 sticky_sessions block is allowed.
* `redirect_http_to_https` - (Optional) A boolean value indicating whether
HTTP requests to the Load Balancer on port 80 will be redirected to HTTPS on port 443.
Default value is `false`.
* `droplet_ids` (Optional) - A list of the IDs of each droplet to be attached to the Load Balancer.
* `droplet_tag` (Optional) - The name of a Droplet tag corresponding to Droplets to be assigned to the Load Balancer.

`forwarding_rule` supports the following:

* `entry_protocol` - (Required) The protocol used for traffic to the Load Balancer. The possible values are: `http`, `https`, or `tcp`.
* `entry_port` - (Required) An integer representing the port on which the Load Balancer instance will listen.
* `target_protocol` - (Required) The protocol used for traffic from the Load Balancer to the backend Droplets. The possible values are: `http`, `https`, or `tcp`.
* `target_port` - (Required) An integer representing the port on the backend Droplets to which the Load Balancer will send traffic.
* `certificate_id` - (Optional) The ID of the TLS certificate to be used for SSL termination.
* `tls_passthrough` - (Optional) A boolean value indicating whether SSL encrypted traffic will be passed through to the backend Droplets. The default value is `false`.

`sticky_sessions` supports the following:

* `type` - (Required) An attribute indicating how and if requests from a client will be persistently served by the same backend Droplet. The possible values are `cookies` or `none`. If not specified, the default value is `none`.
* `cookie_name` - (Optional) The name to be used for the cookie sent to the client. This attribute is required when using `cookies` for the sticky sessions type.
* `cookie_ttl_seconds` - (Optional) The number of seconds until the cookie set by the Load Balancer expires. This attribute is required when using `cookies` for the sticky sessions type.


`healthcheck` supports the following:

* `protocol` - (Required) The protocol used for health checks sent to the backend Droplets. The possible values are `http` or `tcp`.
* `port` - (Optional) An integer representing the port on the backend Droplets on which the health check will attempt a connection.
* `path` - (Optional) The path on the backend Droplets to which the Load Balancer instance will send a request.
* `check_interval_seconds` - (Optional) The number of seconds between between two consecutive health checks. If not specified, the default value is `10`.
* `response_timeout_seconds` - (Optional) The number of seconds the Load Balancer instance will wait for a response until marking a health check as failed. If not specified, the default value is `5`.
* `unhealthy_threshold` - (Optional) The number of times a health check must fail for a backend Droplet to be marked "unhealthy" and be removed from the pool. If not specified, the default value is `3`.
* `healthy_threshold` - (Optional) The number of times a health check must pass for a backend Droplet to be marked "healthy" and be re-added to the pool. If not specified, the default value is `5`.


## Attributes Reference

The following attributes are exported:

* `id` - The ID of the Load Balancer
* `ip`- The ip of the Load Balancer
