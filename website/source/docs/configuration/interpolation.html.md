---
layout: "docs"
page_title: "Interpolation Syntax"
sidebar_current: "docs-config-interpolation"
---

# Interpolation Syntax

Embedded within strings in Terraform, whether you're using the
Terraform syntax or JSON syntax, you can interpolate other values
into strings. These interpolations are wrapped in `${}`, such as
`${var.foo}`.

The interpolation syntax is powerful and allows you to reference
variables, attributes of resources, call functions, etc.

## Available Variables

**To reference user variables**, use the `var.` prefix followed by the
variable name. For example, `${var.foo}` will interpolate the
`foo` variable value. If the variable is a mapping, then you
can reference static keys in the map with the syntax
`var.MAP.KEY`. For example, `${var.amis.us-east-1}` would
get the value of the `us-east-1` key within the `amis` variable
that is a mapping.

**To reference attributes of other resources**, the syntax is
`TYPE.NAME.ATTRIBUTE`. For example, `${aws_instance.web.id}`
will interpolate the ID attribute from the "aws\_instance"
resource named "web".

**To reference outputs from a module**, the syntax is
`MODULE.NAME.OUTPUT`. For example `${module.foo.bar}` will
interpolate the "bar" output from the "foo"
[module](/docs/modules/index.html).

**To reference count information**, the syntax is `count.FIELD`.
For example, `${count.index}` will interpolate the current index
in a multi-count resource. For more information on count, see the
resource configuration page.

## Built-in Functions

Terraform ships with built-in functions. Functions are called with
the syntax `name(arg, arg2, ...)`. For example,
to read a file: `${file("path.txt")}`. The built-in functions
are documented below.

The supported built-in functions are:

  * `concat(args...)` - Concatenates the values of multiple arguments into
      a single string.

  * `file(path)` - Reads the contents of a file into the string. Variables
      in this file are _not_ interpolated. The contents of the file are
      read as-is.

  * `lookup(map, key)` - Performs a dynamic lookup into a mapping
      variable.
