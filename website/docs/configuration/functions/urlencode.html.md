---
layout: "functions"
page_title: "urlencode - Functions - Configuration Language"
sidebar_current: "docs-funcs-encoding-urlencode"
description: |-
  The urlencode function applies URL encoding to a given string.
---

# `urlencode` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`urlencode` applies URL encoding to a given string.

This function identifies characters in the given string that would have a
special meaning when included as a query string argument in a URL and
escapes them using
[RFC 3986 "percent encoding"](https://tools.ietf.org/html/rfc3986#section-2.1).

The exact set of characters escaped may change over time, but the result
is guaranteed to be interpolatable into a query string argument without
inadvertently introducing additional delimiters.

If the given string contains non-ASCII characters, these are first encoded as
UTF-8 and then percent encoding is applied separately to each UTF-8 byte.

## Examples

```
> urlencode("Hello World")
Hello%20World
> urlencode("☃")
%E2%98%83
> "http://example.com/search?q=${urlencode("terraform urlencode")}"
http://example.com/search?q=terraform%20urlencode
```
