---
layout: "scaleway"
page_title: "Scaleway: scaleway_image"
sidebar_current: "docs-scaleway-datasource-image"
description: |-
  Get information on a Scaleway image.
---

# scaleway\_image

Use this data source to get the ID of a registered Image for use with the
`scaleway_server` resource.

## Example Usage

```hcl
data "scaleway_image" "ubuntu" {
  architecture = "arm"
  name         = "Ubuntu Precise"
}

resource "scaleway_server" "base" {
  name  = "test"
  image = "${data.scaleway_image.ubuntu.id}"
  type  = "C1"
}
```

## Argument Reference

* `architecture` - (Required) any supported Scaleway architecture, e.g. `x86_64`, `arm`

* `name_filter` - (Optional) Regexp to match Image name by

* `name` - (Optional) Exact name of desired Image

## Attributes Reference

`id` is set to the ID of the found Image. In addition, the following attributes
are exported:

* `architecture` - architecture of the Image, e.g. `arm` or `x86_64`

* `organization` - uuid of the organization owning this Image

* `public` - is this a public bootscript

* `creation_date` - date when image was created

