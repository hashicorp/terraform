---
layout: "functions"
page_title: "filesha256 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-filesha256"
description: |-
  The filesha256 function computes the SHA256 hash of the contents of
  a given file and encodes it as hex.
---

# `filesha256` Function

-> **Note:** This page is about Terraform 0.12 and later. For Terraform 0.11 and
earlier, see
[0.11 Configuration Language: Interpolation Syntax](../../configuration-0-11/interpolation.html).

`filesha256` is a variant of [`sha256`](./sha256.html)
that hashes the contents of a given file rather than a literal string.

This is similar to `sha256(file(filename))`, but
because [`file`](./file.html) accepts only UTF-8 text it cannot be used to
create hashes for binary files.
