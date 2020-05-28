---
layout: "functions"
page_title: "base64sha384 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-base64sha384"
description: |-
  The base64sha384 function computes the SHA384 hash of a given string and
  encodes it with Base64.
---

# `base64sha384` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`base64sha384` computes the SHA384 hash of a given string and encodes it with
Base64. This is not equivalent to `base64encode(sha384("test"))` since `sha384()`
returns hexadecimal representation. 

The given string is first encoded as UTF-8 and then the SHA384 algorithm is applied
as defined in [RFC 4634](https://tools.ietf.org/html/rfc4634). The raw hash is
then encoded with Base64 before returning. Terraform uses the "standard" Base64
alphabet as defined in [RFC 4648 section 4](https://tools.ietf.org/html/rfc4648#section-4).

## Examples

```
> base64sha384("hello world")
/b2OdaZ/KfcBpOBAOF4uI5hjA+oQI5IRr5B/y7g1eLPkF8txzmRu/QgZ3YwIjeG9
```

## Related Functions

* [`filebase64sha384`](./filebase64sha384.html) calculates the same hash from
  the contents of a file rather than from a string value.
* [`sha384`](./sha384.html) calculates the same hash but returns the result
  in a more-verbose hexadecimal encoding.
