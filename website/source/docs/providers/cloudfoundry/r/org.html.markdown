---
layout: "cloudfoundry"
page_title: "Cloud Foundry: cf_org"
sidebar_current: "docs-cloudfoundry-resource-org"
description: |-
  Provides a Cloud Foundry Org resource.
---

# cf\_org

Provides a Cloud Foundry resource for managing Cloud Foundry [organizations](https://docs.cloudfoundry.org/concepts/roles.html).

## Example Usage

The following example creates an org with two members in addition to users assigned with managers and auditors roles. It is important to define all the members of the org so that Org managers can assign space roles. Each member ID is referenced from resources defined elsewhere in the same Terraform configuration

```
resource "cf_org" "o1" {

    name = "organization-one"

    members = [
        "${cf_user.dev1.id}",
        "${cf_user.dev2.id}" 		
    ]
    managers = [ 
        "${cf_user.manager1.id}" 
    ]
    auditors = [ 
        "${cf_user.auditor1.id}",
		"${cf_user.auditor2.id}" 
    ]

    quota = "${cf_quota.runaway.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Org in Cloud Foundry
* `members` - (Optional) Members of the Org that do not have a specific Org level role but will be assigned to Spaces
* `managers` - (Optional) Members who can administer the Org, such as create spaces and assign roles
* `billing_managers` - (Optional) Members who can create and manage billing account and payment information
* `auditors` - (Optional) Members who can view but cannot edit user information and Org quota usage information
* `quota` - (Optional) The quota or plan to be given to this Org

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the organization
* `quota` - If a quota is not referenced as an argument then the default quota GUID will be exported 
