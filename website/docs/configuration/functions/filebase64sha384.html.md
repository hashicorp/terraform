---
layout: "functions"
page_title: "filebase64sha384 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-filebase64sha384"
description: |-
  The filebase64sha384 function computes the SHA384 hash of the contents of
  a given file and encodes it with Base64.
---

# `filebase64sha384` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`filebase64sha384` is a variant of [`base64sha384`](./base64sha384.html)
that hashes the contents of a given file rather than a literal string.

This is similar to `base64sha384(file(filename))`, but
because [`file`](./file.html) accepts only UTF-8 text it cannot be used to
create hashes for binary files.
