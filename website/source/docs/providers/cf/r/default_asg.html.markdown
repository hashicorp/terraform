---
layout: "cf"
page_title: "Cloud Foundry: cf_default_asg"
sidebar_current: "docs-cf-resource-default-asg"
description: |-
  Provides a Cloud Foundry Default Appliction Security Group resource.
---

# cf\_default\_asg

Provides a resource for modifying the default staging or running
[application security groups](https://docs.cloudfoundry.org/adminguide/app-sec-groups.html).

## Example Usage

The following example shows how to apply [application security groups](http://localhost:4567/docs/providers/cloudfoundry/r/asg.html)
defined elsewhere in the Terraform configuration, to the default running set.  

```
resource "cf_default_asg" "running" {
    name = "running"
    asgs = [ "${cf_asg.messaging.id}", "${cf_asg.services.id}" ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) This should be one of `running` or `staging`
* `asgs` - (Required) A list of references to application security groups IDs.
