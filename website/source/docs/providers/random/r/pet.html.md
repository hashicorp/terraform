---
layout: "random"
page_title: "Random: random_pet"
sidebar_current: "docs-random-resource-pet"
description: |-
  Generates a random pet.
---

# random\_pet

The resource `random_pet` generates random pet names that are intended to be
used as unique identifiers for other resources.

This resource can be used in conjunction with resources that have
the `create_before_destroy` lifecycle flag set, to avoid conflicts with
unique names during the brief period where both the old and new resources
exist concurrently.

## Example Usage

The following example shows how to generate a unique pet name for an AWS EC2
instance that changes each time a new AMI id is selected.

```hcl
resource "random_pet" "server" {
  keepers = {
    # Generate a new pet name each time we switch to a new AMI id
    ami_id = "${var.ami_id}"
  }
}

resource "aws_instance" "server" {
  tags = {
    Name = "web-server-${random_pet.server.id}"
  }

  # Read the AMI id "through" the random_pet resource to ensure that
  # both will change together.
  ami = "${random_pet.server.keepers.ami_id}"

  # ... (other aws_instance arguments) ...
}
```

The result of the above will set the Name of the AWS Instance to
`web-server-simple-snake`.

## Argument Reference

The following arguments are supported:

* `keepers` - (Optional) Arbitrary map of values that, when changed, will
  trigger a new id to be generated. See
  [the main provider documentation](../index.html) for more information.

* `length` - (Optional) The length (in words) of the pet name.

* `prefix` - (Optional) A string to prefix the name with.

* `separator` - (Optional) The character to separate words in the pet name.

## Attribute Reference

The following attributes are supported:

* `id` - (string) The random pet name
