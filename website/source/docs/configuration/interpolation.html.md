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
you to write expressions such as `${count.index + 1}`.

You can escape interpolation with double dollar signs: `$${foo}`
will be rendered as a literal `${foo}`.

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

<a id="path-variables"></a>

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

  * `base64dec(string)` - Given a base64-encoded string, decodes it and
    returns the original string.

  * `base64enc(string)` - Returns a base64-encoded representation of the
    given string.

  * `concat(list1, list2)` - Combines two or more lists into a single list.
     Example: `concat(aws_instance.db.*.tags.Name, aws_instance.web.*.tags.Name)`

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
      `format("web-%03d", count.index + 1)`.

  * `formatlist(format, args...)` - Formats each element of a list
      according to the given format, similarly to `format`, and returns a list.
      Non-list arguments are repeated for each list element.
      For example, to convert a list of DNS addresses to a list of URLs, you might use:
      `formatlist("https://%s:%s/", aws_instance.foo.*.public_dns, var.port)`.
      If multiple args are lists, and they have the same number of elements, then the formatting is applied to the elements of the lists in parallel.
      Example:
      `formatlist("instance %v has private ip %v", aws_instance.foo.*.id, aws_instance.foo.*.private_ip)`.
      Passing lists with different lengths to formatlist results in an error.

  * `index(list, elem)` - Finds the index of a given element in a list. Example:
      `index(aws_instance.foo.*.tags.Name, "foo-test")`

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
      outputs since they currently only support string values. Depending on the
      use, the string this is being performed within may need to be wrapped
      in brackets to indicate that the output is actually a list, e.g.
      `a_resource_param = ["${split(",", var.CSV_STRING)}"]`.
      Example: `split(",", module.amod.server_ids)`

## Templates

Long strings can be managed using templates. [Templates](/docs/providers/template/index.html) are [resources](/docs/configuration/resources.html) defined by a filename and some variables to use during interpolation. They have a computed `rendered` attribute containing the result.

A template resource looks like:

```
resource "template_file" "example" {
    filename = "template.txt"
    vars {
        hello = "goodnight"
        world = "moon"
    }
}

output "rendered" {
    value = "${template_file.example.rendered}"
}
```

Assuming `template.txt` looks like this:

```
${hello} ${world}!
```

Then the rendered value would be `goodnight moon!`.

You may use any of the built-in functions in your template.


### Using Templates with Count

Here is an example that combines the capabilities of templates with the interpolation
from `count` to give us a parametized template, unique to each resource instance:

```
variable "count" {
  default = 2
}

variable "hostnames" {
  default = {
    "0" = "example1.org"
    "1" = "example2.net"
  }
}

resource "template_file" "web_init" {
  // here we expand multiple template_files - the same number as we have instances
  count = "${var.count}"
  filename = "templates/web_init.tpl"
  vars {
    // that gives us access to use count.index to do the lookup
    hostname = "${lookup(var.hostnames, count.index)}"
  }
}

resource "aws_instance" "web" {
  // ...
  count = "${var.count}"
  // here we link each web instance to the proper template_file
  user_data = "${element(template_file.web_init.*.rendered, count.index)}"
}
```

With this, we will build a list of `template_file.web_init` resources which we can
use in combination with our list of `aws_instance.web` resources.

## Math

Simple math can be performed in interpolations:

```
variable "count" {
  default = 2
}

resource "aws_instance" "web" {
  // ...
  count = "${var.count}"

  // tag the instance with a counter starting at 1, ie. web-001
  tags {
    Name = "${format("web-%03d", count.index + 1)}"
  }
}
```

The supported operations are:

- *Add*, *Subtract*, *Multiply*, and *Divide* for **float** types
- *Add*, *Subtract*, *Multiply*, *Divide*, and *Modulo* for **integer** types

-> **Note:** Since Terraform allows hyphens in resource and variable names,
it's best to use spaces between math operators to prevent confusion or unexpected
behavior. For example, `${var.instance-count - 1}` will subtract **1** from the
`instance-count` variable value, while `${var.instance-count-1}` will interpolate
the `instance-count-1` variable value.
