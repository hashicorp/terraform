---
layout: "template"
page_title: "Template: template_dir"
sidebar_current: "docs-template-resource-dir"
description: |-
  Renders templates from a directory.
---

# template_dir

Renders templates from a directory.

## Example Usage
```hcl
data "template_directory" "init" {
  source_dir      = "${path.cwd}/templates"
  destination_dir = "${path.cwd}/templates.generated"
  
  vars {
    consul_address = "${aws_instance.consul.private_ip}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `source_path` - (Required) Path to the directory where the files to template reside.

* `destination_path` - (Required) Path to the directory where the templated files will be written.

* `vars` - (Optional) Variables for interpolation within the template. Note
  that variables must all be primitives. Direct references to lists or maps
  will cause a validation error.

NOTE: Any required parent directories are created automatically. Additionally, any external modification to either the files in the source or destination directories will trigger the resource to be re-created.

## Template Syntax

The syntax of the template files is the same as
[standard interpolation syntax](/docs/configuration/interpolation.html),
but you only have access to the variables defined in the `vars` section.

To access interpolations that are normally available to Terraform
configuration (such as other variables, resource attributes, module
outputs, etc.) you'll have to expose them via `vars` as shown below:

```hcl
resource "template_dir" "init" {
  # ...

  vars {
    foo  = "${var.foo}"
    attr = "${aws_instance.foo.private_ip}"
  }
}
```