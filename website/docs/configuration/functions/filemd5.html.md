---
layout: "functions"
page_title: "filemd5 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-filemd5"
description: |-
  The filemd5 function computes the MD5 hash of the contents of
  a given file and encodes it as hex.
---

# `filemd5` Function

`filemd5` is a variant of [`md5`](./md5.html)
that hashes the contents of a given file rather than a literal string.

This is similar to `md5(file(filename))`, but
because [`file`](./file.html) accepts only UTF-8 text it cannot be used to
create hashes for binary files.
