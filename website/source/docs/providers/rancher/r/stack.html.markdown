---
layout: "rancher"
page_title: "Rancher: rancher_stack"
sidebar_current: "docs-rancher-resource-stack"
description: |-
  Provides a Rancher Stack resource. This can be used to create and manage stacks on rancher.
---

# rancher\_stack

Provides a Rancher Stack resource. This can be used to create and manage stacks on rancher.

## Example Usage

```hcl
# Create a new empty Rancher stack
resource "rancher_stack" "external-dns" {
  name           = "route53"
  description    = "Route53 stack"
  environment_id = "${rancher_environment.default.id}"
  catalog_id     = "library:route53:7"
  scope          = "system"

  environment {
    AWS_ACCESS_KEY        = "MYKEY"
    AWS_SECRET_KEY        = "MYSECRET"
    AWS_REGION            = "eu-central-1"
    TTL                   = "60"
    ROOT_DOMAIN           = "example.com"
    ROUTE53_ZONE_ID       = ""
    HEALTH_CHECK_INTERVAL = "15"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the stack.
* `description` - (Optional) A stack description.
* `environment_id` - (Required) The ID of the environment to create the stack for.
* `docker_compose` - (Optional) The `docker-compose.yml` content to apply for the stack.
* `rancher_compose` - (Optional) The `rancher-compose.yml` content to apply for the stack.
* `environment` - (Optional) The environment to apply to interpret the docker-compose and rancher-compose files.
* `catalog_id` - (Optional) The catalog ID to link this stack to. When provided, `docker_compose` and `rancher_compose` will be retrieved from the catalog unless they are overridden.
* `scope` - (Optional) The scope to attach the stack to. Must be one of **user** or **system**. Defaults to **user**.
* `start_on_create` - (Optional) Whether to start the stack automatically.
* `finish_upgrade` - (Optional) Whether to automatically finish upgrades to this stack.

## Attributes Reference

The following attributes are exported:

* `rendered_docker_compose` - The interpolated `docker_compose` applied to the stack.
* `rendered_rancher_compose` - The interpolated `rancher_compose` applied to the stack.


## Import

Stacks can be imported using the Environment and Stack ID in the form
`<environment_id>/<stack_id>`

```
$ terraform import rancher_stack.foo 1a5/1e149
```

If the credentials for the Rancher provider have access to the global API, then
then `environment_id` can be omitted e.g.

```
$ terraform import rancher_stack.foo 1e149
```
