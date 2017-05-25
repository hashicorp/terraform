---
layout: "rancher"
page_title: "Rancher: rancher_environment"
sidebar_current: "docs-rancher-resource-environment"
description: |-
  Provides a Rancher Environment resource. This can be used to create and manage environments on rancher.
---

# rancher\_environment

Provides a Rancher Environment resource. This can be used to create and manage environments on rancher.

## Example Usage

```hcl
# Create a new Rancher environment
resource "rancher_environment" "default" {
  name = "staging"
  description = "The staging environment"
  orchestration = "cattle"

  member {
    external_id = "650430"
    external_id_type = "github_user"
    role = "owner"
  }

  member {
    external_id = "1234"
    external_id_type = "github_team"
    role = "member"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the environment.
* `description` - (Optional) An environment description.
* `orchestration` - (Optional) Must be one of **cattle**, **swarm**, **mesos** or **kubernetes**. Defaults to **cattle**.
* `member` - (Optional) Members to add to the environment.

### Member Parameters Reference

A `member` takes three parameters:

* `external_id` - (Required) The external ID of the member.
* `external_id_type` - (Required) The external ID type of the member.
* `role` - (Required) The role of the member in the environment.


## Attributes Reference

* `id` - The ID of the environment (ie `1a11`) that can be used in other Terraform resources such as Rancher Stack definitions.

## Import

Environments can be imported using their Rancher API ID, e.g.

```
$ terraform import rancher_environment.dev 1a15
```
