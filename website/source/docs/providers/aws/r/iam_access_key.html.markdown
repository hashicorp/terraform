---
layout: "aws"
page_title: "AWS: aws_iam_access_key"
sidebar_current: "docs-aws-resource-iam-access-key"
description: |-
  Provides an IAM access key. This is a set of credentials that allow API requests to be made as an IAM user.
---

# aws\_iam\_access\_key

Provides an IAM access key. This is a set of credentials that allow API requests to be made as an IAM user.

## Example Usage

```hcl
resource "aws_iam_access_key" "lb" {
  user    = "${aws_iam_user.lb.name}"
  pgp_key = "keybase:some_person_that_exists"
}

resource "aws_iam_user" "lb" {
  name = "loadbalancer"
  path = "/system/"
}

resource "aws_iam_user_policy" "lb_ro" {
  name = "test"
  user = "${aws_iam_user.lb.name}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "ec2:Describe*"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
EOF
}

output "secret" {
  value = "${aws_iam_access_key.lb.encrypted_secret}"
}
```

## Argument Reference

The following arguments are supported:

* `user` - (Required) The IAM user to associate with this access key.
* `pgp_key` - (Optional) Either a base-64 encoded PGP public key, or a
  keybase username in the form `keybase:username`.

## Attributes Reference

The following attributes are exported:

* `id` - The access key ID.
* `user` - The IAM user associated with this access key.
* `key_fingerprint` - The fingerprint of the PGP key used to encrypt
  the secret
* `secret` - The secret access key. Note that this will be written
to the state file. Please supply a `pgp_key` instead, which will prevent the
secret from being stored in plain text
* `encrypted_secret` - The encrypted secret, base64 encoded.
~> **NOTE:** The encrypted secret may be decrypted using the command line,
   for example: `terraform output secret | base64 --decode | keybase pgp decrypt`.
* `ses_smtp_password` - The secret access key converted into an SES SMTP
  password by applying [AWS's documented conversion
  algorithm](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/smtp-credentials.html#smtp-credentials-convert).
* `status` - "Active" or "Inactive". Keys are initially active, but can be made
	inactive by other means.
