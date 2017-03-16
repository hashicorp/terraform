---
layout: "rancher"
page_title: "Rancher: rancher_registration_token"
sidebar_current: "docs-rancher-resource-registration-token"
description: |-
  Provides a Rancher Registration Token resource. This can be used to create registration tokens for rancher environments and retrieve their information.
---

# rancher\_registration\_token

Provides a Rancher Registration Token resource. This can be used to create registration tokens for rancher environments and retrieve their information.

## Example Usage

```hcl
# Create a new Rancher registration token
resource "rancher_registration_token" "default" {
  name           = "staging_token"
  description    = "Registration token for the staging environment"
  environment_id = "${rancher_environment.default.id}"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the registration token.
* `description` - (Optional) A registration token description.
* `environment_id` - (Required) The ID of the environment to create the token for.

## Attributes Reference

The following attributes are exported:

* `command` - The command used to start a rancher agent for this environment.
* `registration_url` - The URL to use to register new nodes to the environment.
* `token` - The token to use to register new nodes to the environment.

## Import

Registration tokens can be imported using the Environment and Registration token
IDs in the form `<environment_id>/<registration_token_id>`.

```
$ terraform import rancher_registration_token.dev_token 1a5/1c11
```

If the credentials for the Rancher provider have access to the global API, then
then `environment_id` can be omitted e.g.

```
$ terraform import rancher_registration_token.dev_token 1c11
```
