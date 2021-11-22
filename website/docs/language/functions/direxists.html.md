---
layout: "language"
page_title: "direxists - Functions - Configuration Language"
sidebar_current: "docs-funcs-file-directory-exists"
description: |-
The direxists function determines whether a directory exists at a given path.
---

# `direxists` Function

`direxists` determines whether a directory exists at a given path.

```hcl
direxists(path)
```

Functions are evaluated during configuration parsing rather than at apply time,
so this function can only be used with directories that are already present on disk
before Terraform takes any actions.

If the path is empty then the result is ".", representing the current working directory.

This function works only with directories. If used with a file, FIFO,
or another file with a special mode, it will return an error.

## Examples

```
> direxists("${path.module}/files")
true
```

## Related Functions

* [`fileexists`](./fileexists.html) determines whether a file exists at a given path.
