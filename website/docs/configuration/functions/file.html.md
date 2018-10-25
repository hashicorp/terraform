---
layout: "functions"
page_title: "file function"
sidebar_current: "docs-funcs-file-file-x"
description: |-
  The file function reads the contents of the file at the given path and
  returns them as a string.
---

# `file` Function

`file` reads the contents of a file at the given path and returns them as
a string.

```hcl
file(path)
```

Strings in the Terraform language are sequences of Unicode characters, so
this function will interpret the file contents as UTF-8 encoded text and
return the resulting Unicode characters. If the file contains invalid UTF-8
sequences then this function will produce an error.

This function can be used only with functions that already exist as static
files on disk at the beginning of a Terraform run. Language functions do not
participate in the dependency graph, so this function cannot be used with
files that are generated dynamically during a Terraform operation. We do not
recommend using of dynamic local files in Terraform configurations, but in rare
situations where this is necessary you can use
[the `local_file` data source](/docs/providers/local/d/file.html)
to read files while respecting resource dependencies.

## Examples

```
> file("${path.module}/hello.txt")
Hello World
```

## Related Functions

* [`filebase64`](./filebase64.html) also reads the contents of a given file,
  but returns the raw bytes in that file Base64-encoded, rather than
  interpreting the contents as UTF-8 text.
* [`fileexists`](./fileexists.html) determines whether a file exists
  at a given path.
