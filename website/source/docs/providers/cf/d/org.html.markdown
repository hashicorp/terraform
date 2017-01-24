---
layout: "cf"
page_title: "Cloud Foundry: cf_org"
sidebar_current: "docs-cf-datasource-org"
description: |-
  Get information on a Cloud Foundry Organization.
---

# cf\_org

Gets information on a Cloud Foundry organization.

## Example Usage

The following example looks up an organization named 'myorg'. 

```
data "cf_org" "o" {
    name = "myorg"    
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the organization to look up

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the organization
