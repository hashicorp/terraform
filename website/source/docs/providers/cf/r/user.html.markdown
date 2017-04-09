---
layout: "cf"
page_title: "Cloud Foundry: cf_user"
sidebar_current: "docs-cf-resource-user"
description: |-
  Provides a Cloud Foundry User resource.
---

# cf\_user

Provides a Cloud Foundry resource for managing users. This resource provides extended 
functionality to attach additional UAA roles to the user.

## Example Usage

The following example creates a user and attaches additional UAA roles to grant administrator rights to that user.

```
resource "cf_user" "admin-service-user" {
    
    name = "cf-admin"
    password = "Passw0rd"
    
    given_name = "John"
    family_name = "Doe"

    groups = [ "cloud_controller.admin", "scim.read", "scim.write" ]

    orgs = [ "${cf_org.org1.id}", "${cf_org.org2.id}" ]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the user. This will also be the users login name
* `password` - (Optional) The user's password
* `origin` - (Optional) The user authentcation origin. By default this will be `UAA`. For users authenticated by LDAP this should be `ldap`
* `given_name` - (Optional) The given name of the user
* `family_name` - (Optional) The family name of the user
* `email` - (Optional) The email address of the user
* `groups` - (Optional) Any UAA `groups` / `roles` to associated the user with
* `orgs` - (Optional) List of [orgs](org.html) this user will be a associated with. This does not assign a role to the user, which needs to be done via the [cf_user_org_role](user_org_role.html.html) resource. However, the user must be associated with the org before he/she can be assigned a role in a [space](space.html) within that org.

## Attributes Reference

The following attributes are exported:

* `id` - The GUID of the User
* `email` - If not provided this attributed will be assigned the same value as the `name`, assuming that the username is the user's email address

