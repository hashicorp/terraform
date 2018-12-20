---
layout: "docs"
page_title: "JSON Configuration Syntax - Configuration Language"
sidebar_current: "docs-config-json-syntax"
description: |-
  In addition to the native syntax that is most commonly used with Terraform,
  the Terraform language can also be expressed in a JSON-compatible syntax.
---

# JSON Configuration Syntax

Most Terraform configurations are written in
[the native Terraform language syntax](./syntax.html), which is designed to be
easy for humans to read and update.

Terraform also supports an alternative syntax that is JSON-compatible. This
syntax is useful when generating portions of a configuration programmatically,
since existing JSON libraries can be used to prepare the generated
configuration files.

The JSON syntax is defined in terms of the native syntax. Everything that can
be expressed in native syntax can also be expressed in JSON syntax, but some
constructs are more complex to represent in JSON due to limitations of the
JSON grammar.

Terraform expects native syntax for files named with a `.tf` suffix, and
JSON syntax for files named with a `.tf.json` suffix.

The low-level JSON syntax, just as with the native syntax, is defined in terms
of a specification called _HCL_. It is not necessary to know all of the details
of HCL syntax or its JSON mapping in order to use Terraform, and so this page
summarizes the most important differences between native and JSON syntax.
If you are interested, you can find a full definition of HCL's JSON syntax
in [its specification](https://github.com/hashicorp/hcl2/blob/master/hcl/json/spec.md).

## JSON File Structure

At the root of any JSON-based Terraform configuration is a JSON object. The
properties of this object correspond to the top-level block types of the
Terraform language. For example:

```json
{
  "variable": {
    "example": {
      "default": "hello"
    }
  }
}
```

Each top-level object property must match the name of one of the expected
top-level block types. Block types that expect labels, such as `variable`
shown above, are represented by one nested object value for each level
of label. `resource` blocks expect two labels, so two levels of nesting
are required:

```json
{
  "resource": {
    "aws_instance": {
      "example": {
        "instance_type": "t2.micro",
        "ami": "ami-abc123"
      }
    }
  }
}
```

After any nested objects representing the labels, finally one more nested
object represents the body of the block itself. In the above examples, the
`default` argument for `variable "example"` and the `instance_type` and
`ami` arguments for `resource "aws_instance" "example"` are specified.

Taken together, the above two configuration files are equivalent to the
following blocks in the native syntax:

```hcl
variable "example" {
  default = "hello"
}

resource "aws_instance" "example" {
  instance_type = "t2.micro"
  ami           = "ami-abc123"
}
```

Within each top-level block type the rules for mapping to JSON are slightly
different (see [Block-type-specific Exceptions][inpage-exceptions] below), but the following general rules apply in most cases:

* The JSON object representing the block body contains properties that
  correspond either to argument names or to nested block type names.

* Where a property corresponds to an argument that accepts
  [arbitrary expressions](./expressions.html) in the native syntax, the
  property value is mapped to an expression as described under
  [_Expression Mapping_](#expression-mapping) below. For arguments that
  do _not_ accept arbitrary expressions, the interpretation of the property
  value depends on the argument, as described in the
  [block-type-specific exceptions](#block-type-specific-exceptions)
  given later in this page.

* Where a property name corresponds to an expected nested block type name,
  the value is interpreted as described under
  [_Nested Block Mapping_](#nested-block-mapping) below, unless otherwise
  stated in [the block-type-specific exceptions](#block-type-specific-exceptions)
  given later in this page.

## Expression Mapping

Since JSON grammar is not able to represent all of the Terraform language
[expression syntax](./expressions.html), JSON values interpreted as expressions
are mapped as follows:

| JSON    | Terraform Language Interpretation                                                                             |
| ------- | ------------------------------------------------------------------------------------------------------------- |
| Boolean | A literal `bool` value.                                                                                       |
| Number  | A literal `number` value.                                                                                     |
| String  | Parsed as a [string template](./expressions.html#string-templates) and then evaluated as described below.     |
| Object  | Each property value is mapped per this table, producing an `object(...)` value with suitable attribute types. |
| Array   | Each element is mapped per this table, producing a `tuple(...)` value with suitable element types.            |
| Null    | A literal `null`.                                                                                             |

When a JSON string is encountered in a location where arbitrary expressions are
expected, its value is first parsed as a [string template](./expressions.html#string-templates)
and then it is evaluated to produce the final result.

If the given template consists _only_ of a single interpolation sequence,
the result of its expression is taken directly, without first converting it
to a string. This allows non-string expressions to be used within the
JSON syntax:

```json
{
  "output": {
    "example": {
      "value": "${aws_instance.example}"
    }
  }
}
```

The `output "example"` declared above has the object value representing the
given `aws_instance` resource block as its value, rather than a string value.
This special behavior does not apply if any literal or control sequences appear
in the template; in these other situations, a string value is always produced.

## Nested Block Mapping

When a JSON object property is named after a nested block type, the value
of this property represents one or more blocks of that type. The value of
the property must be either a JSON object or a JSON array.

The simplest situation is representing only a single block of the given type
when that type expects no labels, as with the `lifecycle` nested block used
within `resource` blocks:

```json
{
  "resource": {
    "aws_instance": {
      "example": {
        "lifecycle": {
          "create_before_destroy": true
        }
      }
    }
  }
}
```

The above is equivalent to the following native syntax configuration:

```hcl
resource "aws_instance" "example" {
  lifecycle {
    create_before_destroy = true
  }
}
```

When the nested block type requires one or more labels, or when multiple
blocks of the same type can be given, the mapping gets a little more
complicated. For example, the `provisioner` nested block type used
within `resource` blocks expects a label giving the provisioner to use,
and the ordering of provisioner blocks is significant to decide the order
of operations.

The following native syntax example shows a `resource` block with a number
of provisioners of different types:

```hcl
resource "aws_instance" "example" {
  # (resource configuration omitted for brevity)

  provisioner "local-exec" {
    command = "echo 'Hello World' >example.txt"
  }
  provisioner "file" {
    source      = "example.txt"
    destination = "/tmp/example.txt"
  }
  provisioner "remote-exec" {
    inline = [
      "sudo install-something -f /tmp/example.txt",
    ]
  }
}
```

In order to preserve the order of these blocks, you must use a JSON array
as the direct value of the property representing this block type, as in
this JSON equivalent of the above:

```json
{
  "resource": {
    "aws_instance": {
      "example": {
        "provisioner": [
          {
            "local-exec": {
              "command": "echo 'Hello World' >example.txt"
            }
          },
          {
            "file": {
              "source": "example.txt",
              "destination": "/tmp/example.txt"
            }
          },
          {
            "remote-exec": {
              "inline": ["sudo install-something -f /tmp/example.txt"]
            }
          }
        ]
      }
    }
  }
}
```

Each element of the `provisioner` array is an object with a single property
whose name represents the label for each `provisioner` block. For block types
that expect multiple labels, this pattern of alternating array and object
nesting can be used for each additional level.

If a nested block type requires labels but the order does _not_ matter, you
may omit the array and provide just a single object whose property names
correspond to unique block labels. This is allowed as a shorthand for the above
for simple cases, but the alternating array and object approach is the most
general. We recommend using the most general form if systematically converting
from native syntax to JSON, to ensure that the meaning of the configuration is
preserved exactly.

### Comment Properties

Although we do not recommend hand-editing of JSON syntax configuration files
-- this format is primarily intended for programmatic generation and consumption --
a limited form of _comments_ are allowed inside JSON objects that represent
block bodies using a special property name:

```json
{
  "resource": {
    "aws_instance": {
      "example": {
        "//": "This instance runs the scheduled tasks for backup",

        "instance_type": "t2.micro",
        "ami": "ami-abc123"
      }
    }
  }
}
```

In any object that represents a block body, properties named `"//"` are
ignored by Terraform entirely. This exception does _not_ apply to objects
that are being [interpreted as expressions](#expression-mapping), where this
would be interpreted as an object type attribute named `"//"`.

This special property name can also be used at the root of a JSON-based
configuration file. This can be useful to note which program created the file.

```json
{
  "//": "This file is generated by generate-outputs.py. DO NOT HAND-EDIT!",

  "output": {
    "example": {
      "value": "${aws_instance.example}"
    }
  }
}
```

## Block-type-specific Exceptions

[inpage-block]: #block-type-specific-exceptions

Certain arguments within specific block types are processed in a special way
by Terraform, and so their mapping to the JSON syntax does not follow the
general rules described above. The following sub-sections describe the special
mapping rules that apply to each top-level block type.

### `resource` and `data` blocks

Some meta-arguments for the `resource` and `data` block types take direct
references to objects, or literal keywords. When represented in JSON, the
reference or keyword is given as a JSON string with no additonal surrounding
spaces or symbols.

For example, the `provider` meta-argument takes a `<PROVIDER>.<ALIAS>` reference
to a provider configuration, which appears unquoted in the native syntax but
must be presented as a string in the JSON syntax:

```json
{
  "resource": {
    "aws_instance": {
      "example": {
        "provider": "aws.foo"
      }
    }
  }
}
```

This special processing applies to the following meta-arguments:

* `provider`: a single string, as shown above
* `depends_on`: an array of strings containing references to named entities,
  like `["aws_instance.example"]`.
* `ignore_changes` within the `lifecycle` block: if set to `all`, a single
  string `"all"` must be given. Otherwise, an array of JSON strings containing
  property references must be used, like `["ami"]`.

Special processing also applies to the `type` argument of any `connection`
blocks, whether directly inside the `resource` block or nested inside
`provisioner` blocks: the given string is interpreted literally, and not
parsed and evaluated as a string template.

### `variable` blocks

All arguments inside `variable` blocks have non-standard mappings to JSON:

* `type`: a string containing a type expression, like `"string"` or `"list(string)"`.
* `default`: a literal JSON value that can be converted to the given type.
  Strings within this value are taken literally and _not_ interpreted as
  string templates.
* `description`: a literal JSON string, _not_ interpreted as a template.

```json
{
  "variable": {
    "example": {
      "type": "string",
      "default": "hello"
    }
  }
}
```

### `output` blocks

The `description` and `sensitive` arguments are interpreted as literal JSON
values. The `description` string is not interpreted as a string template.

The `value` argument is [interpreted as an expression](#expression-mapping).

```json
{
  "output": {
    "example": {
      "value": "${aws_instance.example}"
    }
  }
}
```

### `locals` blocks

The value of the JSON object property representing the locals block type
must be a JSON object whose property names are the local value names to
declare:

```json
{
  "locals": {
    "greeting": "Hello, ${var.name}"
  }
}
```

The value of each of these nested properties is
[interpreted as an expression](#expression-mapping).

### `module` blocks

The `source` and `version` meta-arguments must be given as literal strings. The
values are not interpreted as string templates.

The `providers` meta-argument must be given as a JSON object whose properties
are the compact provider addresses to expose into the child module and whose
values are the provider addresses to use from the current module, both
given as literal strings:

```json
{
  "module": {
    "example": {
      "source": "hashicorp/consul/azurerm",
      "version": "= 1.0.0",
      "providers": {
        "aws": "aws.usw1"
      }
    }
  }
}
```

### `provider` blocks

The `alias` and `version` meta-arguments must be given as literal strings. The
values are not interpreted as string templates.

```json
{
  "provider": {
    "aws": {
      "alias": "usw1",
      "region": "us-west-1"
    }
  }
}
```

### `terraform` blocks

Since no settings within `terraform` blocks accept named object references or
function calls, all setting values are taken literally. String values are not
interpreted as string templates.

Since only one `backend` block is allowed per `terraform` block, the compact
block mapping can be used to represent it, with a nested object containing
a single property whose name represents the backend type.

```json
{
  "terraform": {
    "required_version": ">= 0.12.0",
    "backend": {
      "s3": {
        "region": "us-west-2",
        "bucket": "acme-terraform-states"
      }
    }
  }
}
```
