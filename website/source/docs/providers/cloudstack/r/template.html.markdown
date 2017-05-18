---
layout: "cloudstack"
page_title: "CloudStack: cloudstack_template"
sidebar_current: "docs-cloudstack-resource-template"
description: |-
  Registers an existing template into the CloudStack cloud.
---

# cloudstack_template

Registers an existing template into the CloudStack cloud.

## Example Usage

```hcl
resource "cloudstack_template" "centos64" {
  name       = "CentOS 6.4 x64"
  format     = "VHD"
  hypervisor = "XenServer"
  os_type    = "CentOS 6.4 (64bit)"
  url        = "http://someurl.com/template.vhd"
  zone       = "zone-1"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the template.

* `display_text` - (Optional) The display name of the template.

* `format` - (Required) The format of the template. Valid values are `QCOW2`,
    `RAW`, and `VHD`.

* `hypervisor` - (Required) The target hypervisor for the template. Changing
    this forces a new resource to be created.

* `os_type` - (Required) The OS Type that best represents the OS of this
    template.

* `url` - (Required) The URL of where the template is hosted. Changing this
    forces a new resource to be created.

* `project` - (Optional) The name or ID of the project to create this template for.
    Changing this forces a new resource to be created.

* `zone` - (Required) The name or ID of the zone where this template will be created.
    Changing this forces a new resource to be created.

* `is_dynamically_scalable` - (Optional) Set to indicate if the template contains
    tools to support dynamic scaling of VM cpu/memory (defaults false)

* `is_extractable` - (Optional) Set to indicate if the template is extractable
    (defaults false)

* `is_featured` - (Optional) Set to indicate if the template is featured
    (defaults false)

* `is_public` - (Optional) Set to indicate if the template is available for
    all accounts (defaults true)

* `password_enabled` - (Optional) Set to indicate if the template should be
    password enabled (defaults false)

* `is_ready_timeout` - (Optional) The maximum time in seconds to wait until the
    template is ready for use (defaults 300 seconds)

## Attributes Reference

The following attributes are exported:

* `id` - The template ID.
* `display_text` - The display text of the template.
* `is_dynamically_scalable` - Set to "true" if the template is dynamically scalable.
* `is_extractable` - Set to "true" if the template is extractable.
* `is_featured` - Set to "true" if the template is featured.
* `is_public` - Set to "true" if the template is public.
* `password_enabled` - Set to "true" if the template is password enabled.
* `is_ready` - Set to "true" once the template is ready for use.
