---
layout: "functions"
page_title: "abspath - Functions - Configuration Language"
sidebar_current: "docs-funcs-file-abspath"
description: |-
  The abspath function converts the argument to an absolute filesystem path.
---

# `abspath` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`abspath` takes a string containing a filesystem path and converts it
to an absolute path. That is, if the path is not absolute, it will be joined
with the current working directory.

Referring directly to filesystem paths in resource arguments may cause
spurious diffs if the same configuration is applied from multiple systems or on
different host operating systems. We recommend using filesystem paths only
for transient values, such as the argument to [`file`](./file.html) (where
only the contents are then stored) or in `connection` and `provisioner` blocks.

## Examples

```
> abspath(path.root)
/home/user/some/terraform/root
```
