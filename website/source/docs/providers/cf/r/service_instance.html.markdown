---
layout: "cf"
page_title: "Cloud Foundry: cf_service_instance"
sidebar_current: "docs-cf-resource-service-instance"
description: |-
  Provides a Cloud Foundry Service Instance.
---

# cf\_service_instance

Provides a Cloud Foundry resource for managing Cloud Foundry [Service Instances](https://docs.cloudfoundry.org/devguide/services/) within spaces.

## Example Usage

The following is a Service Instance created within the referenced space and service plan. 

```
resource "cf_service_instance" "redis1" {
	name = "pricing-grid"
    space = "${cf_space.dev.id}"
    servicePlan = "${data.cf_service_plan.redis.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Service Instance in Cloud Foundry
* `servicePlan` - (Required) The ID of the [service plan](http://localhost:4567/docs/providers/cloudfoundry/d/service_plan.html)
* `space` - (Required) The ID of the [space](http://localhost:4567/docs/providers/cloudfoundry/r/space.html) 
* `jsonParameters` - (Optional) List of arbitrary parameters. Some services support providing additional configuration parameters within the provision request
* `tags` - (Optional) List of instance tags. Some services provide a list of tags that Cloud Foundry delivers in [VCAP_SERVICES Env variables](https://docs.cloudfoundry.org/devguide/deploy-apps/environment-variable.html#VCAP-SERVICES)

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the service instance