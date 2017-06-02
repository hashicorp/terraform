---
layout: "template"
page_title: "Template: template_file"
sidebar_current: "docs-template-datasource-file"
description: |-
  Renders a template from a file.
---

# template_file

Renders a template from a file.

## Example Usage

Option 1: From a file:

Reference the template path:

```hcl
data "template_file" "init" {
  template = "${file("${path.module}/init.tpl")}"

  vars {
    consul_address = "${aws_instance.consul.private_ip}"
  }
}
```

Inside the file, reference the variable as such:

```bash
#!/bin/bash

echo "CONSUL_ADDRESS = ${consul_address}" > /tmp/iplist
```

Option 2: Inline:

```hcl
data "template_file" "init" {
  template = "$${consul_address}:1234"

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

* `vars` - (Optional) Variables for interpolation within the template. Note
  that variables must all be primitives. Direct references to lists or maps
  will cause a validation error.

The following arguments are maintained for backwards compatibility and may be
removed in a future version:

* `filename` - _Deprecated, please use `template` instead_. The filename for
  the template. Use [path variables](/docs/configuration/interpolation.html#path-variables) to make
  this path relative to different path roots.

## Attributes Reference

The following attributes are exported:

* `template` - See Argument Reference above.
* `vars` - See Argument Reference above.
* `rendered` - The final rendered template.

## Template Syntax

The syntax of the template files is the same as
[standard interpolation syntax](/docs/configuration/interpolation.html),
but you only have access to the variables defined in the `vars` section.

To access interpolations that are normally available to Terraform
configuration (such as other variables, resource attributes, module
outputs, etc.) you'll have to expose them via `vars` as shown below:

```hcl
data "template_file" "init" {
  # ...

  vars {
    foo  = "${var.foo}"
    attr = "${aws_instance.foo.private_ip}"
  }
}
```

## Inline Templates

Inline templates allow you to specify the template string inline without
loading a file. An example is shown below:

```hcl
data "template_file" "init" {
  template = "$${consul_address}:1234"

  vars {
    consul_address = "${aws_instance.consul.private_ip}"
  }
}
```

-> **Important:** Template variables in an inline template (such as
`consul_address` above) must be escaped with a double-`$`. Unescaped
interpolations will be processed by Terraform normally prior to executing
the template.

An example of mixing escaped and non-escaped interpolations in a template:

```hcl
variable "port" { default = 80 }

data "template_file" "init" {
  template = "$${foo}:${var.port}"

  vars {
    foo = "${count.index}"
  }
}
```

In the above example, the template is processed by Terraform first to
turn it into: `${foo}:80`. After that, the template is processed as a
template to interpolate `foo`.

In general, you should use template variables in the `vars` block and try
not to mix interpolations. This keeps it understandable and has the benefit
that you don't have to change anything to switch your template to a file.
