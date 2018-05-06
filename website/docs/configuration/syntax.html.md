---
layout: "docs"
page_title: "Configuration Syntax"
sidebar_current: "docs-config-syntax"
description: |-
  The Terraform language has its own syntax, intended to combine declarative
  structure with expressions in a way that is easy for humans to read and
  understand.
---

# Configuration Syntax

Other pages in this section have described various configuration constructs
that can appear in the Terraform language. This page describes the lower-level
syntax of the language in more detail, revealing the building blocks that
those constructs are built from.

This page describes the _native syntax_ of the Terraform language, which is
a rich language designed to be easy for humans to read and write. The
constructs in the Terraform language can also be expressed in
[JSON syntax](/docs/configuration/syntax-json.html), which is harder for humans
to read and edit but easier to generate and parse programmatically.

This low-level syntax of the Terraform language is defined in terms of a
syntax called _HCL_, which is also used by configuration languages in
other applications, and in particular other HashiCorp products.
It is not necessary to know all of the details of HCL syntax in
order to use Terraform, and so this page summarizes the most important
details. If you are interested, you can find a full definition of HCL
syntax in
[the HCL native syntax specification](https://github.com/hashicorp/hcl2/blob/master/hcl/hclsyntax/spec.md).

## Attributes and Blocks

The Terraform language syntax is built around two key syntax constructs:
attributes and blocks.

An _attribute_ assigns a value to a particular name:

```hcl
image_id = "abc123"
```

The identifier before the equals sign is the _attribute name_, and after
the equals sign is the attribute's value. The semantics applied to each
attribute name define what value types are valid, but many attributes
accept arbitrary [expressions](/docs/configuration/expressions.html),
which allow the value to either be specified literally or generated from
other values programmatically.

A _block_ is a container for other content:

```hcl
resource "aws_instance" "example" {
  ami = "abc123"

  network_interface {
    # ...
  }
}
```

A block has a _type_ ("resource" in this example). Each block type defines
how many _labels_ must follow the type keyword. The "resource" block type
shown here expects two labels, which are "aws_instance" and "example"
in this case. A particular block type may have any number of required labels,
or it may require none as with the nested "network_interface" block type.

After the block type keyword and any labels, the block _body_ is delimited
by the `{` and `}` characters. Within the block body, further attributes
and blocks may be nested, creating a heirarchy of blocks and their associated
attributes.

Unfortunately, the low-level syntax described here uses the noun "attribute"
to mean something slightly different to how it is used by the main
Terraform language. Elsewhere in this documentation, "attribute" usually
refers to a named value exported by an object that can be accessed in an
expression, such as the "id" portion of the expression
`aws_instance.example.id`. To reduce confusion, other documentation uses the
term "argument" to refer to the syntax-level idea of an attribute.

### Style Conventions

The Terraform parser allows you some flexibility in how you lay out the
elements in your configuration files, but the Terraform language also has some
idiomatic style conventions which we recommend users should always follow
for consistency between files and modules written by different teams.
Automatic source code formatting tools may apply these conventions
automatically.

* Indent two spaces for each nesting level.

* When multiple attributes with single-line values appear on consecutive lines
  at the same nesting level, align their equals signs:

  ```hcl
  ami           = "abc123"
  instance_type = "t2.micro"
  ```

* When both attributes and blocks appear together inside a block body,
  place all of the attributes together at the top and then place nested
  blocks below them. Use one blank line to separate the attributes from
  the blocks.

* Use empty lines to separate logical groups of attributes within a block.

* For blocks that contain both arguments and "meta-arguments" (as defined by
  the Terraform language semantics), list meta-argument attributes first
  and separate them from other attributes with one blank line. Place
  meta-argument blocks _last_ and separate them from other blocks with
  one blank line.

  ```hcl
  resource "aws_instance" "example" {
    count = 2 # meta-argument attribute first

    ami           = "abc123"
    instance_type = "t2.micro"

    network_interface {
      # ...
    }

    lifecycle { # meta-argument block last
      create_before_destroy = true
    }
  }
  ```

* Top-level blocks should always be separated from one another by one
  blank line. Nested blocks should also be separated by blank lines, except
  when grouping together related blocks of the same type.

* Avoid separating multiple blocks of the same type with other blocks of
  a different type, unless the block types are defined by semantics to
  form a family.
  (For example: `root_block_device`, `ebs_block_device` and
  `ephemeral_block_device` on `aws_instance` form a family of block types
  describing AWS block devices, and can therefore be grouped together and
  mixed.)

## Identifiers

Attribute names, block type names, and the names of most Terraform-specific
constructs like resources, input variables, etc. are all _identifiers_.
The Terraform language implements
[the Unicode identifier syntax](http://unicode.org/reports/tr31/), extended
to also include the ASCII hyphen character `-`.

In practice, this means that identifiers can contain letters, digits,
underscores, and hyphens. To avoid ambiguity with literal numbers, the
first character of an identifier must not be a digit.

## Comments

The Terraform language supports three different syntaxes for comments:

* `#` begins a single-line comment, ending at the end of the line

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
