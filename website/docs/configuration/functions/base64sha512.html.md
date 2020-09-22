---
layout: "functions"
page_title: "base64sha512 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-base64sha512"
description: |-
  The base64sha512 function computes the SHA512 hash of a given string and
  encodes it with Base64.
---

# `base64sha512` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`base64sha512` computes the SHA512 hash of a given string and encodes it with
Base64. This is not equivalent to `base64encode(sha512("test"))` since `sha512()`
returns hexadecimal representation. 

The given string is first encoded as UTF-8 and then the SHA512 algorithm is applied
as defined in [RFC 4634](https://tools.ietf.org/html/rfc4634). The raw hash is
then encoded with Base64 before returning. Terraform uses the "standard" Base64
alphabet as defined in [RFC 4648 section 4](https://tools.ietf.org/html/rfc4648#section-4).

## Examples

```
> base64sha512("hello world")
MJ7MSJwS1utMxA9QyQLytNDtd+5RGnx6m808qG1M2G+YndNbxf9JlnDaNCVbRbDP2DDoH2Bdz33FVC6TrpzXbw==
```

## Related Functions

* [`filebase64sha512`](./filebase64sha512.html) calculates the same hash from
  the contents of a file rather than from a string value.
* [`sha512`](./sha512.html) calculates the same hash but returns the result
  in a more-verbose hexadecimal encoding.
