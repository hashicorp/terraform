---
layout: "chef"
page_title: "Chef: chef_role"
sidebar_current: "docs-chef-resource-role"
description: |-
  Creates and manages a role in Chef Server.
---

# chef\_role

A [role](http://docs.chef.io/roles.html) is a set of standard configuration
that can apply across multiple nodes that perform the same function.

## Example Usage

```hcl
resource "chef_role" "example" {
  name     = "example-role"
  run_list = ["recipe[example]"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique name to assign to the role.
* `description` - (Optional) A human-friendly description of the role.
  If not set, a placeholder of "Managed by Terraform" will be set.
* `default_attributes_json` - (Optional) String containing a JSON-serialized
  object containing the default attributes for the role.
* `override_attributes_json` - (Optional) String containing a JSON-serialized
  object containing the override attributes for the role.
* `run_list` - (Optional) List of strings to set as the
  [run list](https://docs.chef.io/run_lists.html) for any nodes that belong
  to this role.

## Attributes Reference

This resource exports no further attributes.
