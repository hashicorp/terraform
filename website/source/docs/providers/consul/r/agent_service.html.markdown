---
layout: "consul"
page_title: "Consul: consul_agent_service"
sidebar_current: "docs-consul-resource-agent-service"
description: |-
  Provides access to Agent Service data in Consul. This can be used to define a service associated with a particular agent. Currently, defining health checks for an agent service is not supported.
---

# consul_agent_service

Provides access to the agent service data in Consul. This can be used to
define a service associated with a particular agent. Currently, defining
health checks for an agent service is not supported.

## Example Usage

```hcl
resource "consul_agent_service" "app" {
  address = "www.google.com"
  name    = "google"
  port    = 80
  tags    = ["tag0", "tag1"]
}
```

## Argument Reference

The following arguments are supported:

* `address` - (Optional) The address of the service. Defaults to the
  address of the agent.

* `name` - (Required) The name of the service.

* `port` - (Optional) The port of the service.

* `tags` - (Optional) A list of values that are opaque to Consul,
  but can be used to distinguish between services or nodes.

## Attributes Reference

The following attributes are exported:

* `address` - The address of the service.
* `id` - The ID of the service, defaults to the value of `name`.
* `name` - The name of the service.
* `port` - The port of the service.
* `tags` - The tags of the service.
