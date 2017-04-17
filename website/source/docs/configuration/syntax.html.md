---
layout: "docs"
page_title: "Configuration Syntax"
sidebar_current: "docs-config-syntax"
description: |-
  The syntax of Terraform configurations is custom. It is meant to strike a
  balance between human readable and editable as well as being machine-friendly.
  For machine-friendliness, Terraform can also read JSON configurations. For
  general Terraform configurations, however, we recommend using the Terraform
  syntax.
---

# Configuration Syntax

The syntax of Terraform configurations is called [HashiCorp Configuration
Language (HCL)](https://github.com/hashicorp/hcl). It is meant to strike a
balance between human readable and editable as well as being machine-friendly.
For machine-friendliness, Terraform can also read JSON configurations. For
general Terraform configurations, however, we recommend using the HCL Terraform
syntax.

## Terraform Syntax

Here is an example of Terraform's HCL syntax:

```hcl
# An AMI
variable "ami" {
  description = "the AMI to use"
}

/* A multi
   line comment. */
resource "aws_instance" "web" {
  ami               = "${var.ami}"
  count             = 2
  source_dest_check = false

  connection {
    user = "root"
  }
}
```

Basic bullet point reference:

  * Single line comments start with `#`

  * Multi-line comments are wrapped with `/*` and `*/`

  * Values are assigned with the syntax of `key = value` (whitespace
    doesn't matter). The value can be any primitive (string,
    number, boolean), a list, or a map.

  * Strings are in double-quotes.

  * Strings can interpolate other values using syntax wrapped
    in `${}`, such as `${var.foo}`. The full syntax for interpolation
    is [documented here](/docs/configuration/interpolation.html).

  * Multiline strings can use shell-style "here doc" syntax, with
    the string starting with a marker like `<<EOF` and then the
    string ending with `EOF` on a line of its own. The lines of
    the string and the end marker must *not* be indented.

  * Numbers are assumed to be base 10. If you prefix a number with
    `0x`, it is treated as a hexadecimal number.

  * Boolean values: `true`, `false`.

  * Lists of primitive types can be made with square brackets (`[]`).
    Example: `["foo", "bar", "baz"]`.

  * Maps can be made with braces (`{}`) and colons (`:`):
    `{ "foo": "bar", "bar": "baz" }`. Quotes may be omitted on keys, unless the
    key starts with a number, in which case quotes are required. Commas are
    required between key/value pairs for single line maps. A newline between
    key/value pairs is sufficient in multi-line maps.

In addition to the basics, the syntax supports hierarchies of sections,
such as the "resource" and "variable" in the example above. These
sections are similar to maps, but visually look better. For example,
these are nearly equivalent:

```hcl
variable "ami" {
  description = "the AMI to use"
}
```

is equal to:

```hcl
variable = [{
  "ami": {
    "description": "the AMI to use",
  }
}]
```

Notice how the top stanza visually looks a lot better? By repeating
multiple `variable` sections, it builds up the `variable` list. When
possible, use sections since they're visually clearer and more readable.

## JSON Syntax

Terraform also supports reading JSON formatted configuration files.
The above example converted to JSON:

```json
{
  "variable": {
    "ami": {
      "description": "the AMI to use"
    }
  },

  "resource": {
    "aws_instance": {
      "web": {
        "ami": "${var.ami}",
        "count": 2,
        "source_dest_check": false,

        "connection": {
          "user": "root"
        }
      }
    }
  }
}
```

The conversion should be pretty straightforward and self-documented.

The downsides of JSON are less human readability and the lack of
comments. Otherwise, the two are completely interoperable.
