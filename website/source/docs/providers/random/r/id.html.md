---
layout: "random"
page_title: "Random: random_id"
sidebar_current: "docs-random-resource-id"
description: |-
  Generates a random identifier.
---

# random\_id

The resource `random_id` generates random numbers that are intended to be
used as unique identifiers for other resources.

Unlike other resources in the "random" provider, this resource *does* use a
cryptographic random number generator in order to minimize the chance of
collisions, making the results of this resource when a 32-byte identifier
is requested of equivalent uniqueness to a type-4 UUID.

This resource can be used in conjunction with resources that have,
the `create_before_destroy` lifecycle flag set, to avoid conflicts with
unique names during the brief period where both the old and new resources
exist concurrently.

## Example Usage

The following example shows how to generate a unique name for an AWS EC2
instance that changes each time a new AMI id is selected.

```hcl
resource "random_id" "server" {
  keepers = {
    # Generate a new id each time we switch to a new AMI id
    ami_id = "${var.ami_id}"
  }

  byte_length = 8
}

resource "aws_instance" "server" {
  tags = {
    Name = "web-server ${random_id.server.hex}"
  }

  # Read the AMI id "through" the random_id resource to ensure that
  # both will change together.
  ami = "${random_id.server.keepers.ami_id}"

  # ... (other aws_instance arguments) ...
}
```

## Argument Reference

The following arguments are supported:

* `byte_length` - (Required) The number of random bytes to produce. The
  minimum value is 1, which produces eight bits of randomness.

* `keepers` - (Optional) Arbitrary map of values that, when changed, will
  trigger a new id to be generated. See
  [the main provider documentation](../index.html) for more information.

* `prefix` - (Optional) Arbitrary string to prefix the output value with. This
  string is supplied as-is, meaning it is not guaranteed to be URL-safe or
  base64 encoded.

## Attributes Reference

The following attributes are exported:

* `b64` - The generated id presented in base64, using the URL-friendly character set: case-sensitive letters, digits and the characters `_` and `-`.
* `hex` - The generated id presented in padded hexadecimal digits. This result will always be twice as long as the requested byte length.
* `dec` - The generated id presented in non-padded decimal digits.
