---
layout: "functions"
page_title: "filemd5 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-filemd5"
description: |-
  The filemd5 function computes the MD5 hash of the contents of
  a given file and encodes it as hex.
---

# `filemd5` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`filemd5` is a variant of [`md5`](./md5.html)
that hashes the contents of a given file rather than a literal string.

This is similar to `md5(file(filename))`, but
because [`file`](./file.html) accepts only UTF-8 text it cannot be used to
create hashes for binary files.
