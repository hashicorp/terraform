---
layout: "rancher"
page_title: "Rancher: rancher_registry_credential"
sidebar_current: "docs-rancher-resource-registry-credential"
description: |-
  Provides a Rancher Registy Credential resource. This can be used to create registry credentials for rancher environments and retrieve their information.
---

# rancher\_registry\_credential

Provides a Rancher Registy Credential resource. This can be used to create registry credentials for rancher environments and retrieve their information.

## Example Usage

```hcl
# Create a new Rancher registry
resource "rancher_registry_credential" "dockerhub" {
  name = "dockerhub"
  description = "DockerHub Registry Credential"
  registry_id = "${rancher_registry.dockerhub.id}"
  email = "myself@company.com"
  public_value = "myself"
  secret_value = "mypass"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the registry credential.
* `description` - (Optional) A registry credential description.
* `registry_id` - (Required) The ID of the registry to create the credential for.
* `email` - (Required) The email of the account.
* `public_value` - (Required) The public value (user name) of the account.
* `secret_value` - (Required) The secret value (password) of the account.

## Attributes Reference

No further attributes are exported.

## Import

Registry credentials can be imported using their Rancher API ID, e.g.

```
$ terraform import rancher_registry_credential.private_registry 1c605
```
