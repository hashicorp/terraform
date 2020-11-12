---
layout: "functions"
page_title: "template - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-replace"
description: |-
  The template function read a string and renders it as template.
---

# `template` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`template` reads a string and renders it as a template using a supplied set of template variables.

```hcl
template(str, vars)
```

The template syntax is the same as for
[string templates](../expressions.html#string-templates) in the main Terraform
language, including interpolation sequences delimited with `${` ... `}`.

The "vars" argument must be a map. Within the template string, each of the keys
in the map is available as a variable for interpolation. The template may
also use any other function available in the Terraform language. Variable names must
each start with a letter, followed by zero or more letters, digits, or
underscores.

In both quoted and heredoc string expressions, Terraform supports template sequences that begin with `${` and `%{`. These are described in more detail in the following section. To include these sequences literally without beginning a template sequence, double the leading character: `$${` or `%%{`.

Strings in the Terraform language are sequences of Unicode characters, so if the string contains invalid UTF-8 sequences then this function will produce an error.

## Examples

The `template` function renders the template:

```
> template("Hello, $${name}!", {name = "Jane"})
Hello, Jane!

```

The `template` function can be used with the `file` function to read a template from a file. Witch behavior is similar to the `templatefile`.

## Related Functions

* [`file`](./file.html) reads a file from disk and returns its literal contents
  without any template interpretation.
