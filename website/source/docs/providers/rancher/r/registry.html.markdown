---
layout: "rancher"
page_title: "Rancher: rancher_registry"
sidebar_current: "docs-rancher-resource-registry"
description: |-
  Provides a Rancher Registy resource. This can be used to create registries for rancher environments and retrieve their information.
---

# rancher\_registry

Provides a Rancher Registy resource. This can be used to create registries for rancher environments and retrieve their information

## Example Usage

```hcl
# Create a new Rancher registry
resource "rancher_registry" "dockerhub" {
  name           = "dockerhub"
  description    = "DockerHub Registry"
  environment_id = "${rancher_environment.default.id}"
  server_address = "index.dockerhub.io"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the registry.
* `description` - (Optional) A registry description.
* `environment_id` - (Required) The ID of the environment to create the registry for.
* `server_address` - (Required) The server address for the registry.

## Attributes Reference

No further attributes are exported.

## Import

Registries can be imported using the Environment and Registry IDs in the form
`<environment_id>/<registry_id>`

```
$ terraform import rancher_registry.private_registry 1a5/1sp31
```

If the credentials for the Rancher provider have access to the global API, then
then `environment_id` can be omitted e.g.

```
$ terraform import rancher_registry.private_registry 1sp31
```
