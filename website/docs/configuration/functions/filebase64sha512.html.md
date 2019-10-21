---
layout: "functions"
page_title: "filebase64sha512 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-filebase64sha512"
description: |-
  The filebase64sha512 function computes the SHA512 hash of the contents of
  a given file and encodes it with Base64.
---

# `filebase64sha512` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`filebase64sha512` is a variant of [`base64sha512`](./base64sha512.html)
that hashes the contents of a given file rather than a literal string.

This is similar to `base64sha512(file(filename))`, but
because [`file`](./file.html) accepts only UTF-8 text it cannot be used to
create hashes for binary files.
