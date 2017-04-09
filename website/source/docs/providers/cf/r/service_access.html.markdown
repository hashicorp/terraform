---
layout: "cf"
page_title: "Cloud Foundry: cf_service_access"
sidebar_current: "docs-cf-resource-service-access"
description: |-
  Provides a Cloud Foundry Service Access resource.
---

# cf\_service\_access

Provides a Cloud Foundry resource for managing [access](https://docs.cloudfoundry.org/services/access-control.html) to service plans published by Cloud Foundry [service brokers](https://docs.cloudfoundry.org/services/).

## Example Usage

The following example enables access to a specific plan of a given service broker within an Org.

```
resource "cf_service_access" "sb" {
    plan = "${cf_service_broker.sb1.services.my-service.plan1}"
    org = "${cf_org.org1.id}"
}
```

## Argument Reference

The following arguments are supported:

* `plan` - (Required) The ID of the service plan to grant access to
* `org` - (Required) The ID of the Org which should have access to the plan
