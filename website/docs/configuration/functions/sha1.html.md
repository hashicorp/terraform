---
layout: "functions"
page_title: "sha1 function"
sidebar_current: "docs-funcs-crypto-sha1"
description: |-
  The sha1 function computes the SHA1 hash of a given string and encodes it
  with hexadecimal digits.
---

# `sha1` Function

`sha1` computes the SHA1 hash of a given string and encodes it with
hexadecimal digits.

The given string is first encoded as UTF-8 and then the SHA1 algorithm is applied
as defined in [RFC 3174](https://tools.ietf.org/html/rfc3174). The raw hash is
then encoded to lowercase hexadecimal digits before returning.

Collision attacks have been successfully performed against this hashing
function. Before using this function for anything security-sensitive, review
relevant literature to understand the security implications.

## Examples

```
> sha1("hello world")
2aae6c35c94fcfb415dbe95f408b9ce91ee846ed
```
