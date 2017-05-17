---
layout: "consul"
page_title: "Consul: consul_catalog_services"
sidebar_current: "docs-consul-data-source-catalog-services"
description: |-
  Provides a list of services in a given Consul datacenter.
---

# consul_catalog_services

The `consul_catalog_services` data source returns a list of Consul services that
have been registered with the Consul cluster in a given datacenter.  By
specifying a different datacenter in the `query_options` it is possible to
retrieve a list of services from a different WAN-attached Consul datacenter.

This data source is different from the `consul_catalog_service` (singular) data
source, which provides a detailed response about a specific Consul service.

## Example Usage

```hcl
data "consul_catalog_services" "read-dc1" {
  query_options {
    # Optional parameter: implicitly uses the current datacenter of the agent  
    datacenter = "dc1"
  }
}

# Set the description to a whitespace delimited list of the services
resource "example_resource" "app" {
  description = "${join(" ", data.consul_catalog_services.names)}"

  # ...
}
```

## Argument Reference

The following arguments are supported:

* `datacenter` - (Optional) The Consul datacenter to query.  Defaults to the
  same value found in `query_options` parameter specified below, or if that is
  empty, the `datacenter` value found in the Consul agent that this provider is
  configured to talk to.

* `query_options` - (Optional) See below.

The `query_options` block supports the following:

* `allow_stale` - (Optional) When `true`, the default, allow responses from
  Consul servers that are followers.

* `require_consistent` - (Optional) When `true` force the client to perform a
  read on at least quorum servers and verify the result is the same.  Defaults
  to `false`.

* `token` - (Optional) Specify the Consul ACL token to use when performing the
  request.  This defaults to the same API token configured by the `consul`
  provider but may be overriden if necessary.

* `wait_index` - (Optional) Index number used to enable blocking quereis.

* `wait_time` - (Optional) Max time the client should wait for a blocking query
  to return.

## Attributes Reference

The following attributes are exported:

* `datacenter` - The datacenter the keys are being read from to.
* `names` - A list of the Consul services found.  This will always contain the
  list of services found.
* `services.<service>` - For each name given, the corresponding attribute is a
  Terraform map of services and their tags.  The value is an alphanumerically
  sorted, whitespace delimited set of tags associated with the service.
* `tags` - A map of the tags found for each service.  If more than one service
  shares the same tag, unique service names will be joined by whitespace (this
  is the inverse of `services` and can be used to lookup the services that match
  a single tag).
