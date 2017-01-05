---
layout: "cf"
page_title: "Cloud Foundry: cloudfoundry_asg"
sidebar_current: "docs-cf-datasource-asg"
description: |-
  Get information on a Cloud Foundry Appliction Security Group.
---

# cf\_asg

Gets information on a Cloud Foundry application security group.

## Example Usage

```
data "cf_asg" "public" {
    name = "public_networks"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the application security group to lookup

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the applicaiton security group
