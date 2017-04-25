---
layout: "scaleway"
page_title: "Scaleway: scaleway_bootscript"
sidebar_current: "docs-scaleway-datasource-bootscript"
description: |-
  Get information on a Scaleway bootscript.
---

# scaleway\_bootscript

Use this data source to get the ID of a registered Bootscript for use with the
`scaleway_server` resource.

## Example Usage

```hcl
data "scaleway_bootscript" "debug" {
  architecture = "arm"
  name_filter  = "Rescue"
}
```

## Argument Reference

* `architecture` - (Optional) any supported Scaleway architecture, e.g. `x86_64`, `arm`

* `name_filter` - (Optional) Regexp to match Bootscript name by

* `name` - (Optional) Exact name of desired Bootscript

## Attributes Reference

`id` is set to the ID of the found Bootscript. In addition, the following attributes
are exported:

* `architecture` - architecture of the Bootscript, e.g. `arm` or `x86_64`

* `organization` - uuid of the organization owning this Bootscript

* `public` - is this a public bootscript

* `boot_cmd_args` - command line arguments used for booting

* `dtb` - path to Device Tree Blob detailing hardware information

* `initrd` - URL to initial ramdisk content

* `kernel` - URL to used kernel

