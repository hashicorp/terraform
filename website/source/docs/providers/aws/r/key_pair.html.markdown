---
layout: "aws"
page_title: "AWS: aws_key_pair"
sidebar_current: "docs-aws-resource-key-pair"
description: |-
  Provides a Key Pair resource. Currently this supports importing an existing key pair but not creating a new key pair.
---

# aws\_key\_pair

Provides an [EC2 key pair](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html) resource. A key pair is used to control login access to EC2 instances.

Currently this resource requires an existing user-supplied key pair. This key pair's public key will be registered with AWS to allow logging-in to EC2 instances.

When importing an existing key pair the public key material may be in any format supported by AWS. Supported formats (per the [AWS documentation](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html#how-to-generate-your-own-key-and-import-it-to-aws)) are:

* OpenSSH public key format (the format in ~/.ssh/authorized_keys)
* Base64 encoded DER format
* SSH public key file format as specified in RFC4716

## Example Usage

```hcl
resource "aws_key_pair" "deployer" {
  key_name   = "deployer-key"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQD3F6tyPEFEzV0LX3X8BsXdMsQz1x2cEikKDEY0aIj41qgxMCP/iteneqXSIFZBp5vizPvaoIR3Um9xK7PGoW8giupGn+EPuxIA4cDM4vzOqOkiMPhz5XK0whEjkVzTo4+S0puvDZuwIsdiW9mxhJc7tgBNL0cYlWSYVkz4G/fslNfRPW5mYAM49f4fhtxPb5ok4Q2Lg9dPKVHO/Bgeu5woMc7RY0p1ej6D4CKFE6lymSDJpW0YHX/wqE9+cfEauh7xZcG0q9t2ta6F6fmX0agvpFyZo8aFbXeUBr7osSCJNgvavWbM/06niWrOvYX2xwWdhXmXSrbX8ZbabVohBK41 email@example.com"
}
```

## Argument Reference

The following arguments are supported:

* `key_name` - (Optional) The name for the key pair.
* `key_name_prefix` - (Optional) Creates a unique name beginning with the specified prefix. Conflicts with `key_name`.
* `public_key` - (Required) The public key material.

## Attributes Reference

The following attributes are exported:

* `key_name` - The key pair name.
* `fingerprint` - The MD5 public key fingerprint as specified in section 4 of RFC 4716.

## Import

Key Pairs can be imported using the `key_name`, e.g.

```
$ terraform import aws_key_pair.deployer deployer-key
```
