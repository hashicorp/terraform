---
layout: "chef"
page_title: "Chef: chef_data_bag"
sidebar_current: "docs-chef-resource-data-bag"
description: |-
  Creates and manages a data bag in Chef Server.
---

# chef\_data\_bag

A [data bag](http://docs.chef.io/data_bags.html) is a collection of
configuration objects that are stored as JSON in Chef Server and can be
retrieved and used in Chef recipes.

This resource creates the data bag itself. Inside each data bag is a collection
of items which can be created using the ``chef_data_bag_item`` resource.

## Example Usage

```
resource "chef_data_bag" "example" {
    name = "example-data-bag"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The unique name to assign to the data bag. This is the
  name that other server clients will use to find and retrieve data from the
  data bag.

## Attributes Reference

The following attributes are exported:

* `api_url` - The URL representing this data bag in the Chef server API.
