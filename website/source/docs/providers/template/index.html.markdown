---
layout: "template"
page_title: "Provider: Template"
sidebar_current: "docs-template-index"
description: |-
  The Template provider is used to template strings for other Terraform resources.
---

# Template Provider

The template provider exposes resources to use templates to generate
strings for other Terraform resources or outputs.

The template provider is what we call a _logical provider_. This has no
impact on how it behaves, but conceptually it is important to understand.
The template provider doesn't manage any _physical_ resources; it isn't
creating servers, writing files, etc. It is used to generate attributes that
can be used for interpolation for other resources. Examples will explain
this best.

Use the navigation to the left to read about the available resources.

## Example Usage

```
# Template for initial configuration bash script
resource "template_file" "init" {
	template = "${file("init.tpl")}"

	vars {
		consul_address = "${aws_instance.consul.private_ip}"
	}
}

# Create a web server
resource "aws_instance" "web" {
    # ...

	user_data = "${template_file.init.rendered}"
}
```
