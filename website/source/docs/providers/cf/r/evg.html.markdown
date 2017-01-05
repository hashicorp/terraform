---
layout: "cf"
page_title: "Cloud Foundry: cf_evg"
sidebar_current: "docs-cf-resource-evg"
description: |-
  Provides a Cloud Foundry Environment Variable Group resource.
---

# cf\_evg

Provides a resource for modifying the running or staging [environment variable groups](https://docs.pivotal.io/pivotalcf/1-8/devguide/deploy-apps/environment-variable.html#evgroups) in Cloud Foundry.

## Example Usage

The example below shows how to add environment variables to the running environment variable group.

```
resource "cf_evg" "running" {

	name = "running"

    variables = {
        name1 = "value1"
        name2 = "value2"
        name3 = "value3"
        name4 = "value4"
    }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Either `running` or `staging` to indicate the type of environment variable group to update
* `variables` - (Required) A map of name-value pairs of environment variables
