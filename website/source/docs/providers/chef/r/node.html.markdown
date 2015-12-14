---
layout: "chef"
page_title: "Chef: chef_node"
sidebar_current: "docs-chef-resource-node"
description: |-
  Creates and manages a node in Chef Server.
---

# chef\_node

A [node](http://docs.chef.io/nodes.html) is a computer whose
configuration is managed by Chef.

Although this resource allows a node to be registered, it does not actually
configure the computer in question to interact with Chef. In most cases it
is better to use [the `chef` provisioner](/docs/provisioners/chef.html) to
configure the Chef client on a computer and have it register itself with the
Chef server.

## Example Usage

```
resource "chef_node" "example" {
    name = "example-environment"
    environment_name = "${chef_environment.example.name}"
    run_list = ["recipe[example]", "role[app_server]"]
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique name to assign to the node.
* `automatic_attributes_json` - (Optional) String containing a JSON-serialized
  object containing the automatic attributes for the node.
* `normal_attributes_json` - (Optional) String containing a JSON-serialized
  object containing the normal attributes for the node.
* `default_attributes_json` - (Optional) String containing a JSON-serialized
  object containing the default attributes for the node.
* `override_attributes_json` - (Optional) String containing a JSON-serialized
  object containing the override attributes for the node.
* `run_list` - (Optional) List of strings to set as the
  [run list](https://docs.chef.io/run_lists.html) for the node.

## Attributes Reference

This resource exports no further attributes.
