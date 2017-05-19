---
layout: "ovh"
page_title: "OVH: publiccloud_user"
sidebar_current: "docs-ovh-resource-publiccloud-user"
description: |-
  Creates a user in a public cloud project.
---

# ovh_publiccloud\_user

Creates a user in a public cloud project.

## Example Usage

```
resource "ovh_publiccloud_user" "user1" {
   project_id = "67890"
}
```

## Argument Reference

The following arguments are supported:

* `project_id` - (Required) The id of the public cloud project. If omitted,
    the `OVH_PROJECT_ID` environment variable is used.

* `description` - A description associated with the user.

## Attributes Reference

The following attributes are exported:

* `project_id` - See Argument Reference above.
* `description` - See Argument Reference above.
* `username` - the username generated for the user. This username can be used with
   the Openstack API.
* `password` - (Sensitive) the password generated for the user. The password can
   be used with the Openstack API. This attribute is sensitive and will only be
   retrieve once during creation.
* `status` - the status of the user. should be normally set to 'ok'.
* `creation_date` - the date the user was created.
* `openstack_rc` - a convenient map representing an openstack_rc file.
   Note: no password nor sensitive token is set in this map.
