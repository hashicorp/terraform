---
layout: "language"
page_title: "templatefile - Functions - Configuration Language"
sidebar_current: "docs-funcs-file-templatefile"
description: |-
  The templatefile function reads the file at the given path and renders its
  content as a template.
---

# `templatefile` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`templatefile` reads the file at the given path and renders its content
as a template using a supplied set of template variables.

```hcl
templatefile(path, vars)
```

The template syntax is the same as for
[string templates](../expressions.html#string-templates) in the main Terraform
language, including interpolation sequences delimited with `${` ... `}`.
This function just allows longer template sequences to be factored out
into a separate file for readability.

The "vars" argument must be a map. Within the template file, each of the keys
in the map is available as a variable for interpolation. The template may
also use any other function available in the Terraform language, except that
recursive calls to `templatefile` are not permitted. Variable names must
each start with a letter, followed by zero or more letters, digits, or
underscores.

Strings in the Terraform language are sequences of Unicode characters, so
this function will interpret the file contents as UTF-8 encoded text and
return the resulting Unicode characters. If the file contains invalid UTF-8
sequences then this function will produce an error.

This function can be used only with files that already exist on disk at the
beginning of a Terraform run. Functions do not participate in the dependency
graph, so this function cannot be used with files that are generated
dynamically during a Terraform operation.

## Examples

### Lists

Given a template file `backends.tpl` with the following content:

```
%{ for addr in ip_addrs ~}
backend ${addr}:${port}
%{ endfor ~}
```

The `templatefile` function renders the template:

```
> templatefile("${path.module}/backends.tpl", { port = 8080, ip_addrs = ["10.0.0.1", "10.0.0.2"] })
backend 10.0.0.1:8080
backend 10.0.0.2:8080

```

### Maps

Given a template file `config.tmpl` with the following content:

```
%{ for config_key, config_value in config }
set ${config_key} = ${config_value}
%{ endfor ~}
```

The `templatefile` function renders the template:

```
> templatefile(
               "${path.module}/backends.tmpl", 
               { 
                 config = {
                   "x"   = "y"
                   "foo" = "bar"
                   "key" = "value"
                 }
               }
              )
set foo = bar
set key = value
set x = y
```

### Generating JSON or YAML from a template

If the string you want to generate will be in JSON or YAML syntax, it's
often tricky and tedious to write a template that will generate valid JSON or
YAML that will be interpreted correctly when using lots of individual
interpolation sequences and directives.

Instead, you can write a template that consists only of a single interpolated
call to either [`jsonencode`](./jsonencode.html) or
[`yamlencode`](./yamlencode.html), specifying the value to encode using
[normal Terraform expression syntax](/docs/configuration/expressions.html)
as in the following examples:

```
${jsonencode({
  "backends": [for addr in ip_addrs : "${addr}:${port}"],
})}
```

```
${yamlencode({
  "backends": [for addr in ip_addrs : "${addr}:${port}"],
})}
```

Given the same input as the `backends.tmpl` example in the previous section,
this will produce a valid JSON or YAML representation of the given data
structure, without the need to manually handle escaping or delimiters.
In the latest examples above, the repetition based on elements of `ip_addrs` is
achieved by using a
[`for` expression](/docs/configuration/expressions.html#for-expressions)
rather than by using
[template directives](/docs/configuration/expressions.html#directives).

```json
{"backends":["10.0.0.1:8080","10.0.0.2:8080"]}
```

If the resulting template is small, you can choose instead to write
`jsonencode` or `yamlencode` calls inline in your main configuration files, and
avoid creating separate template files at all:

```hcl
locals {
  backend_config_json = jsonencode({
    "backends": [for addr in ip_addrs : "${addr}:${port}"],
  })
}
```

For more information, see the main documentation for
[`jsonencode`](./jsonencode.html) and [`yamlencode`](./yamlencode.html).

## Related Functions

* [`file`](./file.html) reads a file from disk and returns its literal contents
  without any template interpretation.
