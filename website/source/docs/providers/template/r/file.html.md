---
layout: "template"
page_title: "Template: template_file"
sidebar_current: "docs-template-resource-file"
description: |-
  Renders a template from a file.
---

# template\_file

Renders a template from a file.

## Example Usage

```
resource "template_file" "init" {
    template = "${file("${path.module}/init.tpl")}"

    vars {
        consul_address = "${aws_instance.consul.private_ip}"
    }
}

```

## Argument Reference

The following arguments are supported:

* `template` - (Required) The contents of the template. These can be loaded
  from a file on disk using the [`file()` interpolation
  function](/docs/configuration/interpolation.html#file_path_).

* `vars` - (Optional) Variables for interpolation within the template.

The following arguments are maintained for backwards compatibility and may be
removed in a future version:

* `filename` - __Deprecated, please use `template` instead_. The filename for
  the template. Use [path variables](/docs/configuration/interpolation.html#path-variables) to make
  this path relative to different path roots.

## Attributes Reference

The following attributes are exported:

* `template` - See Argument Reference above.
* `vars` - See Argument Reference above.
* `rendered` - The final rendered template.

## Template files syntax

The syntax of the template files is [documented here](/docs/configuration/interpolation.html), under the "Templates" section.
