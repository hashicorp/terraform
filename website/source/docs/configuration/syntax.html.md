---
layout: "docs"
page_title: "Configuration Syntax"
sidebar_current: "docs-config-syntax"
description: |-
  The syntax of Terraform configurations is custom. It is meant to strike a balance between human readable and editable as well as being machine-friendly. For machine-friendliness, Terraform can also read JSON configurations. For general Terraform configurations, however, we recommend using the Terraform syntax.
---

# Configuration Syntax

The syntax of Terraform configurations is custom. It is meant to
strike a balance between human readable and editable as well as being
machine-friendly. For machine-friendliness, Terraform can also
read JSON configurations. For general Terraform configurations,
however, we recommend using the Terraform syntax.

## Terraform Syntax

Here is an example of Terraform syntax:

```
# An AMI
variable "ami" {
	description = "the AMI to use"
}

/* A multi
   line comment. */
resource "aws_instance" "web" {
	ami = "${var.ami}"
	count = 2
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
    doesn't matter). The value can be any primitive: a string,
    number, or boolean.

  * Strings are in double-quotes.

  * Strings can interpolate other values using syntax wrapped
    in `${}`, such as `${var.foo}`. The full syntax for interpolation
    is
    [documented here](/docs/configuration/interpolation.html).

  * Multiline strings can use shell-style "here doc" syntax, with
    the string starting with a marker like `<<EOT` and then the
    string ending with `EOT` on a line of its own. The lines of
    the string and the end marker must *not* be indented.

  * Numbers are assumed to be base 10. If you prefix a number with
    `0x`, it is treated as a hexadecimal number.

  * Numbers can be suffixed with `kKmMgG` for some multiple of 10.
    For example: `1k` is equal to `1000`.

  * Numbers can be suffixed with `[kKmMgG]b` for power of 2 multiples,
    example: `1kb` is equal to `1024`.

  * Boolean values: `true`, `false`.

  * Lists of primitive types can be made by wrapping it in `[]`.
    Example: `["foo", "bar", 42]`.

  * Maps can be made with the `{}` syntax:
	`{ "foo": "bar", "bar": "baz" }`.

In addition to the basics, the syntax supports hierarchies of sections,
such as the "resource" and "variable" in the example above. These
sections are similar to maps, but visually look better. For example,
these are nearly equivalent:

```
variable "ami" {
	description = "the AMI to use"
}

# is equal to:

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
