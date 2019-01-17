---
layout: "docs"
page_title: "Local Values - 0.11 Configuration Language"
sidebar_current: "docs-conf-old-locals"
description: |-
  Local values assign a name to an expression that can then be used multiple times
  within a module.
---

# Local Values

-> **Note:** This page is about Terraform 0.11 and earlier. For Terraform 0.12
and later, see
[Configuration Language: Configuring Local Values](../configuration/locals.html).

Local values assign a name to an expression, that can then be used multiple
times within a module.

Comparing modules to functions in a traditional programming language,
if [variables](./variables.html) are analogous to function arguments and
[outputs](./outputs.html) are analogous to function return values then
_local values_ are comparable to a function's local variables.

This page assumes you're already familiar with
[the configuration syntax](/docs/configuration/syntax.html).

## Examples

Local values are defined in `locals` blocks:

```hcl
# Ids for multiple sets of EC2 instances, merged together
locals {
  instance_ids = "${concat(aws_instance.blue.*.id, aws_instance.green.*.id)}"
}

# A computed default name prefix
locals {
  default_name_prefix = "${var.project_name}-web"
  name_prefix         = "${var.name_prefix != "" ? var.name_prefix : local.default_name_prefix}"
}

# Local values can be interpolated elsewhere using the "local." prefix.
resource "aws_s3_bucket" "files" {
  bucket = "${local.name_prefix}-files"
  # ...
}
```

Named local maps can be merged with local maps to implement common or default
values:

```hcl
# Define the common tags for all resources
locals {
  common_tags = {
    Component   = "awesome-app"
    Environment = "production"
  }
}

# Create a resource that blends the common tags with instance-specific tags.
resource "aws_instance" "server" {
  ami           = "ami-123456"
  instance_type = "t2.micro"

  tags = "${merge(
    local.common_tags,
    map(
      "Name", "awesome-app-server",
      "Role", "server"
    )
  )}"
}
```

## Description

The `locals` block defines one or more local variables within a module.
Each `locals` block can have as many locals as needed, and there can be any
number of `locals` blocks within a module.

The names given for the items in the `locals` block must be unique throughout
a module. The given value can be any expression that is valid within
the current module.

The expression of a local value can refer to other locals, but as usual
reference cycles are not allowed. That is, a local cannot refer to itself
or to a variable that refers (directly or indirectly) back to it.

It's recommended to group together logically-related local values into
a single block, particularly if they depend on each other. This will help
the reader understand the relationships between variables. Conversely,
prefer to define _unrelated_ local values in _separate_ blocks, and consider
annotating each block with a comment describing any context common to all
of the enclosed locals.
