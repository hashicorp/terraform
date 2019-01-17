---
layout: "functions"
page_title: "base64decode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-base64decode"
description: |-
  The base64decode function decodes a string containing a base64 sequence.
---

# `base64decode` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`base64decode` takes a string containing a Base64 character sequence and
returns the original string.

Terraform uses the "standard" Base64 alphabet as defined in
[RFC 4648 section 4](https://tools.ietf.org/html/rfc4648#section-4).

Strings in the Terraform language are sequences of unicode characters rather
than bytes, so this function will also interpret the resulting bytes as
UTF-8. If the bytes after Base64 decoding are _not_ valid UTF-8, this function
produces an error.

While we do not recommend manipulating large, raw binary data in the Terraform
language, Base64 encoding is the standard way to represent arbitrary byte
sequences, and so resource types that accept or return binary data will use
Base64 themselves, which avoids the need to encode or decode it directly in
most cases. Various other functions with names containing "base64" can generate
or manipulate Base64 data directly.

## Examples

```
> base64decode("SGVsbG8gV29ybGQ=")
Hello World
```

## Related Functions

* [`base64encode`](./base64encode.html) performs the opposite operation,
  encoding the UTF-8 bytes for a string as Base64.
* [`base64gzip`](./base64gzip.html) applies gzip compression to a string
  and returns the result with Base64 encoding.
* [`filebase64`](./filebase64.html) reads a file from the local filesystem
  and returns its raw bytes with Base64 encoding.
