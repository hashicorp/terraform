---
layout: "functions"
page_title: "filesha512 - Functions - Configuration Language"
sidebar_current: "docs-funcs-crypto-filesha512"
description: |-
  The filesha512 function computes the SHA512 hash of the contents of
  a given file and encodes it as hex.
---

# `filesha512` Function

`filesha512` is a variant of [`sha512`](./sha512.html)
that hashes the contents of a given file rather than a literal string.

This is similar to `sha512(file(filename))`, but
because [`file`](./file.html) accepts only UTF-8 text it cannot be used to
create hashes for binary files.
