---
layout: "cf"
page_title: "Cloud Foundry: cf_service_broker"
sidebar_current: "docs-cf-resource-service-broker"
description: |-
  Provides a Cloud Foundry Service Broker resource.
---

# cf\_service\_broker

Provides a Cloud Foundry resource for managing [service brokers](https://docs.cloudfoundry.org/services/) definitions. 

## Example Usage

The following example creates an service_broker.

```
resource "cf_service_broker" "sb1" {
    name = "my-service-broke"
    url = "https://mysb.cfapps.io"
    username = "mysb_user"
    password = "password"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the service broker
* `url` - (Required) The URL that provides the service broker [API](https://docs.cloudfoundry.org/services/api.html)
* `username` - (Optional) The user name to use to authenticate against the service broker API calls
* `password` - (Optional) The password to authenticate with
* `space` - (Optional) The ID of the space to scope this broker to

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the service broker
* `services_plans` - Map of service plan GUIDs keyed by service "&lt;service name&gt;/&lt;plan name&gt;"
