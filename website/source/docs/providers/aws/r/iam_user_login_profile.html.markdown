---
layout: "aws"
page_title: "AWS: aws_iam_user_login_profile"
sidebar_current: "docs-aws-resource-iam-user-login-profile"
description: |-
  Provides an IAM user login profile and encrypts the password.
---

# aws\_iam\_user\_login\_profile

Provides one-time creation of a IAM user login profile, and uses PGP to
encrypt the password for safe transport to the user. PGP keys can be
obtained from Keybase.

## Example Usage

```hcl
resource "aws_iam_user" "u" {
  name          = "auser"
  path          = "/"
  force_destroy = true
}

resource "aws_iam_user_login_profile" "u" {
  user    = "${aws_iam_user.u.name}"
  pgp_key = "keybase:some_person_that_exists"
}

output "password" {
  value = "${aws_iam_user_login_profile.u.encrypted_password}"
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) The IAM user's name.
* `pgp_key` - (Required) Either a base-64 encoded PGP public key, or a
  keybase username in the form `keybase:username`.
* `password_reset_required` - (Optional, default "true") Whether the
  user should be forced to reset the generated password on first login.
* `password_length` - (Optional, default 20) The length of the generated
  password.

## Attributes Reference

The following attributes are exported:

* `key_fingerprint` - The fingerprint of the PGP key used to encrypt
  the password
* `encrypted_password` - The encrypted password, base64 encoded.

~> **NOTE:** The encrypted password may be decrypted using the command line,
   for example: `terraform output password | base64 --decode | keybase pgp decrypt`.

## Import

IAM Login Profiles may not be imported.
