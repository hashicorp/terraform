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
    filename = "init.tpl"

    vars {
        consul_address = "${aws_instance.consul.private_ip}"
    }
}

```

## Argument Reference

The following arguments are supported:

* `filename` - (Required) The filename for the template. Use path variables
    (documented in the interpolation section) to specify what the path is
    relative to.

* `vars` - (Optional) Variables for interpolation within the template.

## Attributes Reference

The following attributes are exported:

* `filename` - See Argument Reference above.
* `vars` - See Argument Reference above.
* `rendered` - The final rendered template.

## Template files syntax

The syntax of the template files is [documented here](/docs/configuration/interpolation.html), under the "Templates" section.
