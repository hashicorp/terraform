---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_tag"
sidebar_current: "docs-do-resource-tag"
description: |-
  Provides a DigitalOcean Tag resource.
---

# digitalocean\_tag

Provides a DigitalOcean Tag resource. A Tag is a label that can be applied to a
droplet resource in order to better organize or facilitate the lookups and
actions on it. Tags created with this resource can be referenced in your droplet
configuration via their ID or name.

## Example Usage

```
# Create a new SSH key
resource "digitalocean_tag" "default" {
    name = "foobar"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the tag

## Attributes Reference

The following attributes are exported:

* `id` - The name of the tag
* `name` - The name of the tag
