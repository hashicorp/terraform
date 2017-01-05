---
layout: "cf"
page_title: "Cloud Foundry: cf_quota"
sidebar_current: "docs-cf-resource-quota"
description: |-
  Provides a Cloud Foundry Quota resource.
---

# cf\_quota

Provides a Cloud Foundry resource to manage [quotas](https://docs.cloudfoundry.org/adminguide/quota-plans.html) 
that can be applied to Orgs and Spaces.

## Example Usage

The following example creates a quota that can be applied to an Org.

```
resource "cf_quota" "large" {
    name = "large"
    allow_paid_service_plans = false
    instance_memory = 2048
    total_memory = 51200
    total_app_instances = 100
    total_routes = 50
    total_services = 200
    total_route_ports = 5
}
```

The following example creates a quota within an Org referenced by `cf_org.myorg.id` that can be applied to any space within that Org.

```
resource "cf_quota" "10g" {
    name = "10g"
    allow_paid_service_plans = false
    instance_memory = 512
    total_memory = 10240
    total_app_instances = 10
    total_routes = 5
    total_services = 20
    org = "${cf_org.myorg.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name you use to identify the quota or plan in Cloud Foundry
* `allow_paid_service_plans` - (Required) Determines whether users can provision instances of non-free service plans. Does not control plan visibility. When false, non-free service plans may be visible in the marketplace but instances can not be provisioned.
* `instance_memory` - (Optional) Maximum memory per application instance
* `total_memory` - (Required) Maximum memory usage allowed
* `total_app_instances` - (Optional) Maximum app instances allowed
* `total_routes` - (Required) Maximum routes allowed
* `total_services` - (Required) Maximum services allowed
* `total_route_ports` - (Optional) Maximum routes with reserved ports
* `total_private_domains` - (Optional) Maximum number of private domains allowed to be created within the Org
* `org` - (Optional) The Org within which this quota will be defined so it can be applied to spaces within that Org

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the quota
