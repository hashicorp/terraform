---
layout: "functions"
page_title: "hextobase64 - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-hextobase64"
description: |-
  The hextobase64 function decodes a hexadecimal string and then
  converts the result to a base64 representation
---

# `hextobase64` Function

`hextobase64` takes a valid hex encoded string, decodes the 
value to its binary representation and then converts to a base64
encoded representation which is then returned.

While we do not recommend manipulating large, raw binary data in the Terraform
language, some cloud resources return hexadecmial strings which may need 
converting to base64 representations.

## Examples

```
> hextobase64("69206c6f7665207465727261666f726d")
aSBsb3ZlIHRlcnJhZm9ybQ==
> base64decode("aSBsb3ZlIHRlcnJhZm9ybQ==")
i love terraform
```