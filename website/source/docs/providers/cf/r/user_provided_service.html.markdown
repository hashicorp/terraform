---
layout: "cf"
page_title: "Cloud Foundry: cf_user_provided_service"
sidebar_current: "docs-cf-resource-user-provided-service"
description: |-
  Provides a Cloud Foundry User Provided Service.
---

# cf\_user_provided_service

Provides a Cloud Foundry resource for managing Cloud Foundry [User Provided Services](https://docs.cloudfoundry.org/devguide/services/user-provided.html) within spaces.

## Example Usage

The following is a User Provided Service created within the referenced space. 

```
resource "cf_user_provided_service" "mq" {
	name = "mq-server"
    space = "${cf_space.dev.id}"
    credentials = {
		  "url" = "mq://localhost:9000"
		  "username" = "admin"
		  "password" = "admin"    
	  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Service Instance in Cloud Foundry
* `space` - (Required) The ID of the [space](http://localhost:4567/docs/providers/cloudfoundry/r/space.html) 
* `credentials` - (Optional) Arbitrary credentials in the form of key-value pairs and delivered to applications via [VCAP_SERVICES Env variables](https://docs.cloudfoundry.org/devguide/deploy-apps/environment-variable.html#VCAP-SERVICES)
* `syslogDrainURL` - (Optional) URL to which logs for bound applications will be streamed
* `routeServiceURL` - (Optional) URL to which requests for bound routes will be forwarded. Scheme for this URL must be https

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the service instance