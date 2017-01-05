---
layout: "cf"
page_title: "Cloud Foundry: cf_quota"
sidebar_current: "docs-cf-datasource-quota"
description: |-
  Get information on a Cloud Foundry Quota.
---

# cf\_quota

Gets information on a Cloud Foundry quota.

## Example Usage

The following example looks up a quota named 'myquota' within the Org identified by the id of an Org resource defined elsewhere in the Terraform configuration. 

```
data "cf_quota" "q" {
    name = "myquota"
    org = "${cf_org.o1.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the quota to look up
* `org` - (Optional) The organization within which the quota is defined

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the quota
