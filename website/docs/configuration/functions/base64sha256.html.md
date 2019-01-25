---
layout: "functions"
page_title: "base64sha256 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-base64sha256"
description: |-
  The base64sha256 function computes the SHA256 hash of a given string and
  encodes it with Base64.
---

# `base64sha256` Function

`base64sha256` computes the SHA256 hash of a given string and encodes it with
Base64. This is not equivalent to base64encode(sha256512("test")) since sha512() 
returns hexadecimal representation. 

The given string is first encoded as UTF-8 and then the SHA256 algorithm is applied
as defined in [RFC 4634](https://tools.ietf.org/html/rfc4634). The raw hash is
then encoded with Base64 before returning. Terraform uses the "standard" Base64
alphabet as defined in [RFC 4648 section 4](https://tools.ietf.org/html/rfc4648#section-4).

## Examples

```
> base64sha256("hello world")
uU0nuZNNPgilLlLX2n2r+sSE7+N6U4DukIj3rOLvzek=
```

## Related Functions

* [`filebase64sha256`](./filebase64sha256.html) calculates the same hash from
  the contents of a file rather than from a string value.
* [`sha256`](./sha256.html) calculates the same hash but returns the result
  in a more-verbose hexadecimal encoding.
