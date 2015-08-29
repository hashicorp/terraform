---
layout: "chef"
page_title: "Chef: chef_environment"
sidebar_current: "docs-chef-resource-environment"
description: |-
  Creates and manages an environment in Chef Server.
---

# chef\_environment

An [environment](http://docs.chef.io/environments.html) is a container for
Chef nodes that share a set of attribute values and may have a set of version
constraints for which cookbook versions may be used on its nodes.

## Example Usage

```
resource "chef_environment" "example" {
    name = "example-environment"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique name to assign to the environment. This name
  will be used when nodes are created within the environment.
* `description` - (Optional) A human-friendly description of the environment.
  If not set, a placeholder of "Managed by Terraform" will be set.
* `default_attributes_json` - (Optional) String containing a JSON-serialized
  object containing the default attributes for the environment.
* `override_attributes_json` - (Optional) String containing a JSON-serialized
  object containing the override attributes for the environment.
* `cookbook_constraints` - (Optional) Mapping of cookbook names to cookbook
  version constraints that should apply for this environment.

## Attributes Reference

This resource exports no further attributes.
