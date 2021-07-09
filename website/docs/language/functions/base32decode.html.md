---
layout: "language"
page_title: "base32decode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-base32decode"
description: |-
    The base32decode function decodes a string containing a base32 sequence.
---

# `base32decode` Function

`base32decode` takes a string containing a Base32 character sequence and
returns the original string.

Terraform uses the "standard" Base32 alphabet as defined in
[RFC 4648 section 6](https://tools.ietf.org/html/rfc4648#section-6).

Strings in the Terraform language are sequences of unicode characters rather
than bytes, so this function will also interpret the resulting bytes as
UTF-8. If the bytes after Base32 decoding are _not_ valid UTF-8, this function
produces an error.

While we do not recommend manipulating large, raw binary data in the Terraform
language, Base32 encoding is the standard way to represent arbitrary byte
sequences, and so resource types that accept or return binary data will use
Base32 themselves, which avoids the need to encode or decode it directly in
most cases. Various other functions with names containing "base32" can generate
or manipulate Base32 data directly.

`base32decode` is, in effect, a shorthand for calling
[`textdecodebase32`](./textdecodebase32.html) with the encoding name set to
`UTF-8`.

## Examples

```
> base32decode("JBSWY3DPEB3W64TMMQ======")
Hello world
```

## Related Functions

-   [`base32encode`](./base32encode.html) performs the opposite operation,
    encoding the UTF-8 bytes for a string as Base32.
-   [`textdecodebase32`](./textdecodebase32.html) is a more general function that
    supports character encodings other than UTF-8.
