---
layout: "functions"
page_title: "fileexists - Functions - Configuration Language"
sidebar_current: "docs-funcs-file-file-exists"
description: |-
  The fileexists function determines whether a file exists at a given path.
---

# `fileexists` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`fileexists` determines whether a file exists at a given path.

```hcl
fileexists(path)
```

Functions are evaluated during configuration parsing rather than at apply time,
so this function can only be used with files that are already present on disk
before Terraform takes any actions.

This function works only with regular files. If used with a directory, FIFO,
or other special mode, it will return an error.

## Examples

```
> fileexists("${path.module}/hello.txt")
true
```

```hcl
fileexists("custom-section.sh") ? file("custom-section.sh") : local.default_content
```

## Related Functions

* [`file`](./file.html) reads the contents of a file at a given path
