---
layout: "template"
page_title: "Template: template_dir"
sidebar_current: "docs-template-resource-dir"
description: |-
  Renders a directory of templates.
---

# template_dir

Renders a directory containing templates into a separate directory of
corresponding rendered files.

`template_dir` is similar to [`template_file`](../d/file.html) but it walks
a given source directory and treats every file it encounters as a template,
rendering it to a corresponding file in the destination directory.

~> **Note** When working with local files, Terraform will detect the resource
as having been deleted each time a configuration is applied on a new machine
where the destination dir is not present and will generate a diff to create
it. This may cause "noise" in diffs in environments where configurations are
routinely applied by many different users or within automation systems.

## Example Usage

The following example shows how one might use this resource to produce a
directory of configuration files to upload to a compute instance, using
Amazon EC2 as a placeholder.

```hcl
resource "template_dir" "config" {
  source_dir      = "${path.module}/instance_config_templates"
  destination_dir = "${path.cwd}/instance_config"
  
  vars {
    consul_addr = "${var.consul_addr}"
  }
}

resource "aws_instance" "server" {
  ami           = "${var.server_ami}"
  instance_type = "t2.micro"

  connection {
    # ...connection configuration...
  }

  provisioner "file" {
    # Referencing the template_dir resource ensures that it will be
    # created or updated before this aws_instance resource is provisioned.
    source      = "${template_dir.config.destination_dir}"
    destination = "/etc/myapp"
  }
}

variable "consul_addr" {}

variable "server_ami" {}
```

## Argument Reference

The following arguments are supported:

* `source_dir` - (Required) Path to the directory where the files to template reside.

* `destination_dir` - (Required) Path to the directory where the templated files will be written.

* `vars` - (Optional) Variables for interpolation within the template. Note
  that variables must all be primitives. Direct references to lists or maps
  will cause a validation error.

Any required parent directories of `destination_dir` will be created
automatically, and any pre-existing file or directory at that location will
be deleted before template rendering begins.

After rendering this resource remembers the content of both the source and
destination directories in the Terraform state, and will plan to recreate the
output directory if any changes are detected during the plan phase.

Note that it is _not_ safe to use the `file` interpolation function to read
files create by this resource, since that function can be evaluated before the
destination directory has been created or updated. It *is* safe to use the
generated files with resources that directly take filenames as arguments,
as long as the path is constructed using the `destination_dir` attribute
to create a dependency relationship with the `template_dir` resource.

## Template Syntax

The syntax of the template files is the same as
[standard interpolation syntax](/docs/configuration/interpolation.html),
but you only have access to the variables defined in the `vars` section.

To access interpolations that are normally available to Terraform
configuration (such as other variables, resource attributes, module
outputs, etc.) you can expose them via `vars` as shown below:

```hcl
resource "template_dir" "init" {
  # ...

  vars {
    foo  = "${var.foo}"
    attr = "${aws_instance.foo.private_ip}"
  }
}
```

## Attributes

This resource exports the following attributes:

* `destination_dir` - The destination directory given in configuration.
  Interpolate this attribute into other resource configurations to create
  a dependency to ensure that the destination directory is populated before
  another resource attempts to read it.
