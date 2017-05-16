---
layout: "opc"
page_title: "Oracle: opc_compute_image_list_entry"
sidebar_current: "docs-opc-resource-image-list-entry"
description: |-
  Creates and manages an Image List Entry in an OPC identity domain.
---

# opc\_compute\_image\_list_entry

The ``opc_compute_image_list_entry`` resource creates and manages an Image List Entry in an OPC identity domain.

## Example Usage

```hcl
resource "opc_compute_image_list" "test" {
  name        = "imagelist1"
  description = "This is a description of the Image List"
  default     = 21
}

resource "opc_compute_image_list_entry" "test" {
  name           = "${opc_compute_image_list.test.name}"
  machine_images = [ "/oracle/public/oel_6.7_apaas_16.4.5_1610211300" ]
  version        = 1
  attributes     = <<JSON
{
  "foo": "bar"
}
JSON
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Image List.

* `machine_images` - (Required) An array of machine images.

* `version` - (Required) The unique version of the image list entry, as an integer.

* `attributes` - (Optional) JSON String of optional data that will be passed to an instance of this machine image when it is launched.

## Attributes Reference

In addition to the above arguments, the following attributes are exported

* `uri` - The Unique Resource Identifier for the Image List Entry.

## Import

Image List's can be imported using the `resource name`, e.g.

```shell
$ terraform import opc_compute_image_list_entry.entry1 example
```
