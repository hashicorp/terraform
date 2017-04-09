---
layout: "cf"
page_title: "Cloud Foundry: cf_user_org_role"
sidebar_current: "docs-cf-resource-user-org-role"
description: |-
  Provides a Cloud Foundry User Org Role resource.
---

# cf\_user\_org\_role

Provides a Cloud Foundry resource for managing a users's roles in orgs. To be able to 
access an Org's resources including Spaces, the user needs to have and Org role defined.

## Example Usage

The following example associates a user with specific roles in multiple orgs.

```
resource "cf_user_org_role" "some-user" {
    user = "${cf_user.some-user.id}"

    role {
      type = "manager"
      org = "${cf_org.org1.id}"
    }
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) The ID of the user to be associated.
* `role` - (Required) List of org roles to grant the user when associated within the give org.
  
  - `type` - (Options) The role type can take one of the following values.

      + ***manager***: Org Managers are managers or other users who need to administer the org.
      + ***billing_manager***:  Org Billing Managers create and manage billing account and payment information.
      + ***auditor***: Org Auditors view but cannot edit user information and org quota usage information.
      + ***member***: (Default) A member of the Org with no specific access rights. These user need to be granted a role by an Org Manager. If the user will only have a Space role then he/she needs to be associated with the Org using this role before a Space role can be assigned.

  - `org` - (Required) The Org ID to associated
