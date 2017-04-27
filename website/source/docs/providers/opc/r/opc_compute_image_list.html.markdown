---
layout: "opc"
page_title: "Oracle: opc_compute_image_list"
sidebar_current: "docs-opc-resource-image-list-type"
description: |-
  Creates and manages an Image List in an OPC identity domain.
---

# opc\_compute\_image\_list

The ``opc_compute_image_list`` resource creates and manages an Image List in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_image_list" "test" {
  name        = "imagelist1"
  description = "This is a description of the Image List"
  default     = 21
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Image List.

* `description` - (Required) A description of the Image List.

* `default` - (Required) The image list entry to be used, by default, when launching instances using this image list. Defaults to `1`.

## Import

Image List's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_image_list.imagelist1 example
```
