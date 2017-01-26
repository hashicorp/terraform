---
layout: "cf"
page_title: "Cloud Foundry: cf_service_plan"
sidebar_current: "docs-cf-datasource-service-plan"
description: |-
  Get information on a Cloud Foundry Service Plan.
---

# cf\_service

Gets information on a Cloud Foundry service plan.

## Example Usage

The following example looks up a service plan named 'shared-vm' within a service identified by its id. 

```
data "cf_service_plan" "redis" {
    name = "shared-vm"
    service = "${cf_service.redis.id}"    
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the service to look up
* `service` - (Required) The service within which the service plan is defined

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the service plan
