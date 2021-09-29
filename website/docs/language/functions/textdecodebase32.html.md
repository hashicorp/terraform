---
layout: "language"
page_title: "textdecodebase32 - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-textdecodebase32"
description: |-
    The textdecodebase32 function decodes a string that was previously Base32-encoded,
    and then interprets the result as characters in a specified character encoding.
---

# `textdecodebase32` Function

-> **Note:** This function is supported only in Terraform v0.14 and later.

`textdecodebase32` function decodes a string that was previously Base32-encoded,
and then interprets the result as characters in a specified character encoding.

Terraform uses the "standard" Base32 alphabet as defined in
[RFC 4648 section 6](https://datatracker.ietf.org/doc/html/rfc4648#section-6).

The `encoding_name` argument must contain one of the encoding names or aliases
recorded in
[the IANA character encoding registry](https://www.iana.org/assignments/character-sets/character-sets.xhtml).
Terraform supports only a subset of the registered encodings, and the encoding
support may vary between Terraform versions.

Terraform accepts the encoding name `UTF-8`, which will produce the same result
as [`base32decode`](./base32decode.html).

## Examples

```
> textdecodebase32("JAAGKADMABWAA3YAEAAFOADPABZAA3AAMQAA====", "UTF-16LE")
Hello World
```

## Related Functions

-   [`textencodebase32`](./textencodebase32.html) performs the opposite operation,
    applying target encoding and then Base32 to a string.
-   [`base32decode`](./base32decode.html) is effectively a shorthand for
    `textdecodebase32` where the character encoding is fixed as `UTF-8`.
