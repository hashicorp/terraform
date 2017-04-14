---
layout: "chef"
page_title: "Chef: chef_data_bag_item"
sidebar_current: "docs-chef-resource-data-bag-item"
description: |-
  Creates and manages an object within a data bag in Chef Server.
---

# chef_data_bag_item

A [data bag](http://docs.chef.io/data_bags.html) is a collection of
configuration objects that are stored as JSON in Chef Server and can be
retrieved and used in Chef recipes.

This resource creates objects within an existing data bag. To create the
data bag itself, use the ``chef_data_bag`` resource.

## Example Usage

```hcl
resource "chef_data_bag_item" "example" {
  data_bag_name = "example-data-bag"

  content_json = <<EOT
{
    "id": "example-item",
    "any_arbitrary_data": true
}
EOT
}
```

## Argument Reference

The following arguments are supported:

* `data_bag_name` - (Required) The name of the data bag into which this item
  will be placed.
* `content_json` - (Required) A string containing a JSON object that will be
  the content of the item. Must at minimum contain a property called "id"
  that is unique within the data bag, which will become the identifier of
  the created item.

## Attributes Reference

The following attributes are exported:

* `id` - The value of the "id" property in the ``content_json`` JSON object,
  which can be used by clients to retrieve this item's content.
