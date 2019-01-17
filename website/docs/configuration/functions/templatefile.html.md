---
layout: "functions"
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
recursive calls to `templatefile` are not permitted.

Strings in the Terraform language are sequences of Unicode characters, so
this function will interpret the file contents as UTF-8 encoded text and
return the resulting Unicode characters. If the file contains invalid UTF-8
sequences then this function will produce an error.

This function can be used only with files that already exist on disk at the
beginning of a Terraform run. Functions do not participate in the dependency
graph, so this function cannot be used with files that are generated
dynamically during a Terraform operation. We do not recommend using dynamic
templates in Terraform configurations, but in rare situations where this is
necessary you can use
[the `template_file` data source](/docs/providers/template/d/file.html)
to render templates while respecting resource dependencies.

## Examples

Given a template file `backends.tmpl` with the following content:

```
%{ for addr in ip_addrs ~}
backend ${addr}:${port}
%{ endfor ~}
```

The `templatefile` function renders the template:

```
> templatefile("${path.module}/backends.tmpl", { port = 8080, ip_addrs = ["10.0.0.1", "10.0.0.2"] })
backend 10.0.0.1:8080
backend 10.0.0.2:8080

```

## Related Functions

* [`file`](./file.html) reads a file from disk and returns its literal contents
  without any template interpretation.
