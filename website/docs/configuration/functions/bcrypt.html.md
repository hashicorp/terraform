---
layout: "functions"
page_title: "bcrypt function"
sidebar_current: "docs-funcs-crypto-bcrypt"
description: |-
  The bcrypt function computes a hash of the given string using the Blowfish
  cipher.
---

# `bcrypt` Function

`bcrypt` computes a hash of the given string using the Blowfish cipher,
returning a string in
[the _Modular Crypt Format_](https://passlib.readthedocs.io/en/stable/modular_crypt_format.html)
usually expected in the shadow password file on many Unix systems.

```hcl
bcrypt(string, cost)
```

The `cost` argument is optional and will default to 10 if unspecified.

Since a bcrypt hash value includes a randomly selected salt, each call to this
function will return a different value, even if the given string and cost are
the same. Using this function directly with resource arguments will therefore
cause spurious diffs. We recommend using this function only in `provisioner`
blocks, or in data resources whose results are only used in `provisioner`
blocks.

The version prefix on the generated string (e.g. `$2a$`) may change in future
versions of Terraform.

## Examples

```
> bcrypt("hello world")
$2a$10$D5grTTzcsqyvAeIAnY/mYOIqliCoG7eAMX0/oFcuD.iErkksEbcAa
```
