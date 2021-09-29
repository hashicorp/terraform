---
layout: "language"
page_title: "base32encode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-base32encode"
description: |-
    The base32encode function applies Base32 encoding to a string.
---

# `base32encode` Function

`base32encode` applies Base32 encoding to a string.

Terraform uses the "standard" Base32 alphabet as defined in
[RFC 4648 section 6](https://datatracker.ietf.org/doc/html/rfc4648#section-6).

Strings in the Terraform language are sequences of unicode characters rather
than bytes, so this function will first encode the characters from the string
as UTF-8, and then apply Base32 encoding to the result.

The Terraform language applies Unicode normalization to all strings, and so
passing a string through `base32decode` and then `base32encode` may not yield
the original result exactly.

While we do not recommend manipulating large, raw binary data in the Terraform
language, Base32 encoding is the standard way to represent arbitrary byte
sequences, and so resource types that accept or return binary data will use
Base32 themselves, and so this function exists primarily to allow string
data to be easily provided to resource types that expect Base32 bytes.

`base32encode` is, in effect, a shorthand for calling
[`textencodebase32`](./textencodebase32.html) with the encoding name set to
`UTF-8`.

## Examples

```
> base32encode("Hello world")
JBSWY3DPEB3W64TMMQ======
```

## Related Functions

-   [`base32decode`](./base32decode.html) performs the opposite operation,
    decoding Base32 data and interpreting it as a UTF-8 string.
-   [`textencodebase32`](./textencodebase32.html) is a more general function that
    supports character encodings other than UTF-8.
