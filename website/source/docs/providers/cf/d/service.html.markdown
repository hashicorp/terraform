---
layout: "cf"
page_title: "Cloud Foundry: cf_service"
sidebar_current: "docs-cf-datasource-service"
description: |-
  Get information on a Cloud Foundry Service.
---

# cf\_service

Gets information on a Cloud Foundry service.

## Example Usage

The following example looks up a service named 'p-redis' globally. 

```
data "cf_service" "redis" {
    name = "p-redis"    
}
```

The following example looks up a service named 'p-redis' within the Space identified by the id of an Space resource defined elsewhere in the Terraform configuration

```
data "cf_service" "redis" {
    name = "p-redis"  
    space = "${cf_space.dev.id}"  
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the service to look up
* `space` - (Optional) The space within which the service is defined

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the service
