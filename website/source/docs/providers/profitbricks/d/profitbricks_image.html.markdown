---
layout: "profitbricks"
page_title: "ProfitBricks : profitbrick_image"
sidebar_current: "docs-profitbricks-datasource-image"
description: |-
  Get information on a ProfitBricks Images
---

# profitbricks\_image

The images data source can be used to search for and return an existing image which can then be used to provision a server.

## Example Usage

```hcl
data "profitbricks_image" "image_example" {
  name     = "Ubuntu"
  type     = "HDD"
  version  = "14"
  location = "location_id"
}
```

## Argument Reference

 * `name` - (Required) Name or part of the name of an existing image that you want to search for.
 * `version` - (Optional) Version of the image (see details below).
 * `location` - (Optional) Id of the existing image's location.
 * `type` - (Optional) The image type, HDD or CD-ROM.

If both "name" and "version" are provided the plugin will concatenate the two strings in this format [name]-[version].

## Attributes Reference

 * `id` - UUID of the image
