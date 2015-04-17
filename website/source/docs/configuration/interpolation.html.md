---
layout: "docs"
page_title: "Interpolation Syntax"
sidebar_current: "docs-config-interpolation"
description: |-
  Embedded within strings in Terraform, whether you're using the Terraform syntax or JSON syntax, you can interpolate other values into strings. These interpolations are wrapped in `${}`, such as `${var.foo}`.
---

# Interpolation Syntax

Embedded within strings in Terraform, whether you're using the
Terraform syntax or JSON syntax, you can interpolate other values
into strings. These interpolations are wrapped in `${}`, such as
`${var.foo}`.

The interpolation syntax is powerful and allows you to reference
variables, attributes of resources, call functions, etc.

You can also perform simple math in interpolations, allowing
you to write expressions such as `${count.index+1}`.

## Available Variables

**To reference user variables**, use the `var.` prefix followed by the
variable name. For example, `${var.foo}` will interpolate the
`foo` variable value. If the variable is a mapping, then you
can reference static keys in the map with the syntax
`var.MAP.KEY`. For example, `${var.amis.us-east-1}` would
get the value of the `us-east-1` key within the `amis` variable
that is a mapping.

**To reference attributes of your own resource**, the syntax is
`self.ATTRIBUTE`. For example `${self.private_ip_address}` will
interpolate that resource's private IP address. Note that this is
only allowed/valid within provisioners.

**To reference attributes of other resources**, the syntax is
`TYPE.NAME.ATTRIBUTE`. For example, `${aws_instance.web.id}`
will interpolate the ID attribute from the "aws\_instance"
resource named "web". If the resource has a `count` attribute set,
you can access individual attributes with a zero-based index, such
as `${aws_instance.web.0.id}`. You can also use the splat syntax
to get a list of all the attributes: `${aws_instance.web.*.id}`.
This is documented in more detail in the
[resource configuration page](/docs/configuration/resources.html).

**To reference outputs from a module**, the syntax is
`MODULE.NAME.OUTPUT`. For example `${module.foo.bar}` will
interpolate the "bar" output from the "foo"
[module](/docs/modules/index.html).

**To reference count information**, the syntax is `count.FIELD`.
For example, `${count.index}` will interpolate the current index
in a multi-count resource. For more information on count, see the
resource configuration page.

**To reference path information**, the syntax is `path.TYPE`.
TYPE can be `cwd`, `module`, or `root`. `cwd` will interpolate the
cwd. `module` will interpolate the path to the current module. `root`
will interpolate the path of the root module. In general, you probably
want the `path.module` variable.

## Built-in Functions

Terraform ships with built-in functions. Functions are called with
the syntax `name(arg, arg2, ...)`. For example,
to read a file: `${file("path.txt")}`. The built-in functions
are documented below.

The supported built-in functions are:

  * `concat(args...)` - Concatenates the values of multiple arguments into
      a single string.

  * `element(list, index)` - Returns a single element from a list
      at the given index. If the index is greater than the number of
      elements, this function will wrap using a standard mod algorithm.
      A list is only possible with splat variables from resources with
      a count greater than one.
      Example: `element(aws_subnet.foo.*.id, count.index)`

  * `file(path)` - Reads the contents of a file into the string. Variables
      in this file are _not_ interpolated. The contents of the file are
      read as-is.

  * `format(format, args...)` - Formats a string according to the given
      format. The syntax for the format is standard `sprintf` syntax.
      Good documentation for the syntax can be [found here](http://golang.org/pkg/fmt/).
      Example to zero-prefix a count, used commonly for naming servers:
      `format("web-%03d", count.index+1)`.

  * `join(delim, list)` - Joins the list with the delimiter. A list is
      only possible with splat variables from resources with a count
      greater than one. Example: `join(",", aws_instance.foo.*.id)`

  * `length(list)` - Returns a number of members in a given list
      or a number of characters in a given string.
      * `${length(split(",", "a,b,c"))}` = 3
      * `${length("a,b,c")}` = 5

  * `lookup(map, key)` - Performs a dynamic lookup into a mapping
      variable. The `map` parameter should be another variable, such
      as `var.amis`.

  * `replace(string, search, replace)` - Does a search and replace on the
      given string. All instances of `search` are replaced with the value
      of `replace`. If `search` is wrapped in forward slashes, it is treated
      as a regular expression. If using a regular expression, `replace`
      can reference subcaptures in the regular expression by using `$n` where
      `n` is the index or name of the subcapture. If using a regular expression,
      the syntax conforms to the [re2 regular expression syntax](https://code.google.com/p/re2/wiki/Syntax).

  * `split(delim, string)` - Splits the string previously created by `join`
      back into a list. This is useful for pushing lists through module
      outputs since they currently only support string values.
      Example: `split(",", module.amod.server_ids)`
