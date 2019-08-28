---
layout: "functions"
page_title: "fileset - Functions - Configuration Language"
sidebar_current: "docs-funcs-file-file-set"
description: |-
  The fileset function enumerates a set of regular file names given a pattern.
---

# `fileset` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`fileset` enumerates a set of regular file names given a path and pattern.
The path is automatically removed from the resulting set of file names and any
result still containing path separators always returns forward slash (`/`) as
the path separator for cross-system compatibility.

```hcl
fileset(path, pattern)
```

Supported pattern matches:

- `*` - matches any sequence of non-separator characters
- `?` - matches any single non-separator character
- `[RANGE]` - matches a range of characters
- `[^RANGE]` - matches outside the range of characters

Functions are evaluated during configuration parsing rather than at apply time,
so this function can only be used with files that are already present on disk
before Terraform takes any actions.

## Examples

```
> fileset(path.module, "files/*.txt")
[
  "files/hello.txt",
  "files/world.txt",
]

> fileset("${path.module}/files", "*.txt")
[
  "hello.txt",
  "world.txt",
]
```

A common use of `fileset` is to create one resource instance per matched file, using
[the `for_each` meta-argument](/docs/configuration/resources.html#for_each-multiple-resource-instances-defined-by-a-map-or-set-of-strings):

```hcl
resource "example_thing" "example" {
  for_each = fileset(path.module, "files/*")

  # other configuration using each.value
}
```
