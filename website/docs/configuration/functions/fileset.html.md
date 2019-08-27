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

`fileset` enumerates a set of regular file names given a pattern.

```hcl
fileset(pattern)
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
> fileset("${path.module}/*.txt")
[
  "path/to/module/hello.txt",
  "path/to/module/world.txt",
]
```

```hcl
resource "example_thing" "example" {
  for_each = fileset("${path.module}/files/*")

  # other configuration using each.value
}
```
