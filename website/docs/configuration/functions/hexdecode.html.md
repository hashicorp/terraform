---
layout: "functions"
page_title: "hexdecode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-hexdecode"
description: |-
  The hexdecode function decodes a hexadecimal string.
---

# `hexdecode` Function

`hexdecode` takes a string containing a valid hex encoded string and
returns the decoded value string.

While we do not recommend manipulating large, raw binary data in the Terraform
language, some cloud resources return hexadecmial strings and may require
further processing.

## Examples

```
> hexdecode("69206c6f7665207465727261666f726d")
i love terraform
```