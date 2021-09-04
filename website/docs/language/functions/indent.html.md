---
layout: "language"
page_title: "indent - Functions - Configuration Language"
sidebar_current: "docs-funcs-string-indent"
description: |-
  The indent function adds a number of spaces to the beginnings of all but the
  first line of a given multi-line string.
---

# `indent` Function

`indent` adds a given number of spaces to the beginnings of all but the first
line in a given multi-line string.

```hcl
indent(num_spaces, string)
```

## Examples

This function is useful for inserting a multi-line string into an
already-indented context in another string:

```
> "  items: ${indent(2, "[\n  foo,\n  bar,\n]")}"
  items: [
    foo,
    bar,
  ]
```

The first line of the string is not indented so that, as above, it can be
placed after an introduction sequence that has already begun the line.

Note that whitespaces are added even after the final newline character.
If your mutli-line string ends with newline character (e.g.: here-doc literal),
use [`chomp`](./chomp.html) function to avoid it:

```tf
locals {
  is_bad_user         = <<-EOS
    user_id IS NULL
    OR user_id == ''
  EOS
  where_without_chomp = "NOT (\n  ${indent(2, local.is_bad_user)})\n"
  where_with_chomp    = "NOT (\n  ${indent(2, chomp(local.is_bad_user))}\n)\n"
}
```

The result will be:

```txt
NOT (
  user_id IS NULL
  OR user_id == ''
  )
```

vs.

```txt
NOT (
  user_id IS NULL
  OR user_id == ''
)
```
