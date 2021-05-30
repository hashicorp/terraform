---
layout: "language"
page_title: "defaults - Functions - Configuration Language"
sidebar_current: "docs-funcs-conversion-defaults"
description: |-
  The defaults function can fill in default values in place of null values.
---

# `defaults` Function

-> **Note:** This function is available only in Terraform 0.15 and later.

~> **Experimental:** This function is part of
[the optional attributes experiment](/docs/language/expressions/type-constraints.html#experimental-optional-object-type-attributes)
and is only available in modules where the `module_variable_optional_attrs`
experiment is explicitly enabled.

The `defaults` function is a specialized function intended for use with
input variables whose type constraints are object types or collections of
object types that include optional attributes.

When you define an attribute as optional and the caller doesn't provide an
explicit value for it, Terraform will set the attribute to `null` to represent
that it was omitted. If you want to use a placeholder value other than `null`
when an attribute isn't set, you can use the `defaults` function to concisely
assign default values only where an attribute value was set to `null`.

```
defaults(input_value, defaults)
```

The `defaults` function expects that the `input_value` argument will be the
value of an input variable with an exact [type constraint](/docs/language/expressions/types.html)
(not containing `any`). The function will then visit every attribute in
the data structure, including attributes of nested objects, and apply the
default values given in the defaults object.

The interpretation of attributes in the `defaults` argument depends on what
type an attribute has in the `input_value`:

* **Primitive types** (`string`, `number`, `bool`): if a default value is given
  then it will be used only if the `input_value`'s attribute of the same
  name has the value `null`. The default value's type must match the input
  value's type.
* **Structural types** (`object` and `tuple` types): Terraform will recursively
  visit all of the attributes or elements of the nested value and repeat the
  same defaults-merging logic one level deeper. The default value's type must
  be of the same kind as the input value's type, and a default value for an
  object type must only contain attribute names that appear in the input
  value's type.
* **Collection types** (`list`, `map`, and `set` types): Terraform will visit
  each of the collection elements in turn and apply defaults to them. In this
  case the default value is only a single value to be applied to _all_ elements
  of the collection, so it must have a type compatible with the collection's
  element type rather than with the collection type itself.

The above rules may be easier to follow with an example. Consider the following
Terraform configuration:

```hcl
terraform {
  # Optional attributes and the defaults function are
  # both experimental, so we must opt in to the experiment.
  experiments = [module_variable_optional_attrs]
}

variable "storage" {
  type = object({
    name    = string
    enabled = optional(bool)
    website = object({
      index_document = optional(string)
      error_document = optional(string)
    })
    documents = map(
      object({
        source_file  = string
        content_type = optional(string)
      })
    )
  })
}

locals {
  storage = defaults(var.storage, {
    # If "enabled" isn't set then it will default
    # to true.
    enabled = true

    # The "website" attribute is required, but
    # it's here to provide defaults for the
    # optional attributes inside.
    website = {
      index_document = "index.html"
      error_document = "error.html"
    }

    # The "documents" attribute has a map type,
    # so the default value represents defaults
    # to be applied to all of the elements in
    # the map, not for the map itself. Therefore
    # it's a single object matching the map
    # element type, not a map itself.
    documents = {
      # If _any_ of the map elements omit
      # content_type then this default will be
      # used instead.
      content_type = "application/octet-stream"
    }
  })
}

output "storage" {
  value = local.storage
}
```

To test this out, we can create a file `terraform.tfvars` to provide an example
value for `var.storage`:

```hcl
storage = {
  name = "example"

  website = {
    error_document = "error.txt"
  }
  documents = {
    "index.html" = {
      source_file  = "index.html.tmpl"
      content_type = "text/html"
    }
    "error.txt" = {
      source_file  = "error.txt.tmpl"
      content_type = "text/plain"
    }
    "terraform.exe" = {
      source_file  = "terraform.exe"
    }
  }
}
```

The above value conforms to the variable's type constraint because it only
omits attributes that are declared as optional. Terraform will automatically
populate those attributes with the value `null` before evaluating anything
else, and then the `defaults` function in `local.storage` will substitute
default values for each of them.

The result of this `defaults` call would therefore be the following object:

```
storage = {
  "documents" = tomap({
    "error.txt" = {
      "content_type" = "text/plain"
      "source_file"  = "error.txt.tmpl"
    }
    "index.html" = {
      "content_type" = "text/html"
      "source_file"  = "index.html.tmpl"
    }
    "terraform.exe" = {
      "content_type" = "application/octet-stream"
      "source_file"  = "terraform.exe"
    }
  })
  "enabled" = true
  "name" = "example"
  "website" = {
    "error_document" = "error.txt"
    "index_document" = "index.html"
  }
}
```

Notice that `enabled` and `website.index_document` were both populated directly
from the defaults. Notice also that the `"terraform.exe"` element of
`documents` had its `content_type` attribute populated from the `documents`
default, but the default value didn't need to predict that there would be an
element key `"terraform.exe"` because the default values apply equally to
all elements of the map where the optional attributes are `null`.

## Using `defaults` elsewhere

The design of the `defaults` function depends on input values having
well-specified type constraints, so it can reliably recognize the difference
between similar types: maps vs. objects, lists vs. tuples. The type constraint
causes Terraform to convert the caller's value to conform to the constraint
and thus `defaults` can rely on the input to conform.

Elsewhere in the Terraform language it's typical to be less precise about
types, for example using the object construction syntax `{ ... }` to construct
values that will be used as if they are maps. Because `defaults` uses the
type information of `input_value`, an `input_value` that _doesn't_ originate
in an input variable will tend not to have an appropriate value type and will
thus not be interpreted as expected by `defaults`.

We recommend using `defaults` only with fully-constrained input variable values
in the first argument, so you can use the variable's type constraint to
explicitly distinguish between collection and structural types.
