---
layout: "template"
page_title: "Provider: Template"
sidebar_current: "docs-template-index"
description: |-
  The Template provider is used to template strings for other Terraform resources.
---

# Template Provider

The template provider exposes data sources to use templates to generate
strings for other Terraform resources or outputs.

Use the navigation to the left to read about the available data sources.

## Example Usage

```hcl
# Template for initial configuration bash script
data "template_file" "init" {
  template = "${file("init.tpl")}"

  vars {
    consul_address = "${aws_instance.consul.private_ip}"
  }
}

# Create a web server
resource "aws_instance" "web" {
  # ...

  user_data = "${data.template_file.init.rendered}"
}
```

Or using an inline template:

```hcl
# Template for initial configuration bash script
data "template_file" "init" {
  template = "$${consul_address}:1234"

  vars {
    consul_address = "${aws_instance.consul.private_ip}"
  }
}

# Create a web server
resource "aws_instance" "web" {
  # ...

  user_data = "${data.template_file.init.rendered}"
}
```

-> **Note:** Inline templates must escape their interpolations (as seen
by the double `$` above). Unescaped interpolations will be processed
_before_ the template.
