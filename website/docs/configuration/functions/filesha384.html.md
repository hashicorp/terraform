---
layout: "functions"
page_title: "filesha384 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-filesha384"
description: |-
  The filesha384 function computes the SHA384 hash of the contents of
  a given file and encodes it as hex.
---

# `filesha384` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`filesha384` is a variant of [`sha384`](./sha384.html)
that hashes the contents of a given file rather than a literal string.

This is similar to `sha384(file(filename))`, but
because [`file`](./file.html) accepts only UTF-8 text it cannot be used to
create hashes for binary files.
