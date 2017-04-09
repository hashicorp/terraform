---
layout: "cf"
page_title: "Cloud Foundry: cf_org"
sidebar_current: "docs-cf-resource-org"
description: |-
  Provides a Cloud Foundry Org resource.
---

# cf\_org

Provides a Cloud Foundry resource for managing Cloud Foundry [organizations](https://docs.cloudfoundry.org/concepts/roles.html). To associate users with specific org roles use the [cf_user_org_role](user_org_role.html) resource.

## Example Usage

The following example creates an org with a specific org-wide quota.

```
resource "cf_org" "o1" {
    name = "organization-one"
    quota = "${cf_quota.runaway.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Org in Cloud Foundry
* `quota` - (Optional) The quota or plan to be given to this Org

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the organization
* `quota` - If a quota is not referenced as an argument then the default quota GUID will be exported 
