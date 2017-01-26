---
layout: "cf"
page_title: "Cloud Foundry: cf_org"
sidebar_current: "docs-cf-datasource-space"
description: |-
  Get information on a Cloud Foundry Space.
---

# cf\_space

Gets information on a Cloud Foundry space.

## Example Usage

The following example looks up a space named 'myspace' within an organization 'myorg' which has been previously provisioned thru Terraform. 

```
data "cf_space" "s" {
    name = "myspace"
    org = "${cf_org.myorg.id}"    
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the space to look up
* `org` - (Required) GUID of the organization 

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the space
* `name` - The name of the space 
* `org` - The GUID of the org it belongs to
* `quota`- The GUID of the space's quota

