---
layout: "docs"
page_title: "Syntax - Configuration Language"
sidebar_current: "docs-config-syntax"
description: |-
  The Terraform language has its own syntax, intended to combine declarative
  structure with expressions in a way that is easy for humans to read and
  understand.
---

# Configuration Syntax

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Syntax](../configuration-0-11/syntax.html).

Other pages in this section have described various configuration constructs
that can appear in the Terraform language. This page describes the lower-level
syntax of the language in more detail, revealing the building blocks that
those constructs are built from.

This page describes the _native syntax_ of the Terraform language, which is
a rich language designed to be easy for humans to read and write. The
constructs in the Terraform language can also be expressed in
[JSON syntax](./syntax-json.html), which is harder for humans
to read and edit but easier to generate and parse programmatically.

This low-level syntax of the Terraform language is defined in terms of a
syntax called _HCL_, which is also used by configuration languages in
other applications, and in particular other HashiCorp products.
It is not necessary to know all of the details of HCL syntax in
order to use Terraform, and so this page summarizes the most important
details. If you are interested, you can find a full definition of HCL
syntax in
[the HCL native syntax specification](https://github.com/hashicorp/hcl2/blob/master/hcl/hclsyntax/spec.md).

## Arguments and Blocks

The Terraform language syntax is built around two key syntax constructs:
arguments and blocks.

### Arguments

An _argument_ assigns a value to a particular name:

```hcl
image_id = "abc123"
```

The identifier before the equals sign is the _argument name_, and the expression
after the equals sign is the argument's value.

The context where the argument appears determines what value types are valid
(for example, each resource type has a schema that defines the types of its
arguments), but many arguments accept arbitrary
[expressions](./expressions.html), which allow the value to
either be specified literally or generated from other values programmatically.

-> **Note:** Terraform's configuration language is based on a more general
language called HCL, and HCL's documentation usually uses the word "attribute"
instead of "argument." These words are similar enough to be interchangeable in
this context, and experienced Terraform users might use either term in casual
conversation. But because Terraform also interacts with several _other_ things
called "attributes" (in particular, Terraform resources have attributes like
`id` that can be referenced from expressions but can't be assigned values in
configuration), we've chosen to use "argument" in the Terraform documentation
when referring to this syntax construct.

### Blocks

A _block_ is a container for other content:

```hcl
resource "aws_instance" "example" {
  ami = "abc123"

  network_interface {
    # ...
  }
}
```

A block has a _type_ (`resource` in this example). Each block type defines
how many _labels_ must follow the type keyword. The `resource` block type
expects two labels, which are `aws_instance` and `example` in the example above.
A particular block type may have any number of required labels, or it may
require none as with the nested `network_interface` block type.

After the block type keyword and any labels, the block _body_ is delimited
by the `{` and `}` characters. Within the block body, further arguments
and blocks may be nested, creating a heirarchy of blocks and their associated
arguments.

The Terraform language uses a limited number of _top-level block types,_ which
are blocks that can appear outside of any other block in a configuration file.
Most of Terraform's features (including resources, input variables, output
values, data sources, etc.) are implemented as top-level blocks.

## Identifiers

Argument names, block type names, and the names of most Terraform-specific
constructs like resources, input variables, etc. are all _identifiers_.

Identifiers can contain letters, digits, underscores (`_`), and hyphens (`-`).
The first character of an identifier must not be a digit, to avoid ambiguity
with literal numbers.

For complete identifier rules, Terraform implements
[the Unicode identifier syntax](http://unicode.org/reports/tr31/), extended to
include the ASCII hyphen character `-`.

## Comments

The Terraform language supports three different syntaxes for comments:

* `#` begins a single-line comment, ending at the end of the line.
* `//` also begins a single-line comment, as an alternative to `#`.
* `/*` and `*/` are start and end delimiters for a comment that might span
  over multiple lines.

The `#` single-line comment style is the default comment style and should be
used in most cases. Automatic configuration formatting tools may automatically
transform `//` comments into `#` comments, since the double-slash style is
not idiomatic.

## Character Encoding and Line Endings

Terraform configuration files must always be UTF-8 encoded. While the
delimiters of the language are all ASCII characters, Terraform accepts
non-ASCII characters in identifiers, comments, and string values.

Terraform accepts configuration files with either Unix-style line endings
(LF only) or Windows-style line endings (CR then LF), but the idiomatic style
is to use the Unix convention, and so automatic configuration formatting tools
may automatically transform CRLF endings to LF.
