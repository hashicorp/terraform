---
layout: "random"
page_title: "Provider: Random"
sidebar_current: "docs-random-index"
description: |-
  The Random provider is used to generate randomness.
---

# Random Provider

The "random" provider allows the use of randomness within Terraform
configurations. This is a *logical provider*, which means that it works
entirely within Terraform's logic, and doesn't interact with any other
services.

Unconstrained randomness within a Terraform configuration would not be very
useful, since Terraform's goal is to converge on a fixed configuration by
applying a diff. Because of this, the "random" provider provides an idea of
*managed randomness*: it provides resources that generate random values during
their creation and then hold those values steady until the inputs are changed.

Even with these resources, it is advisable to keep the use of randomness within
Terraform configuration to a minimum, and retain it for special cases only;
Terraform works best when the configuration is well-defined, since its behavior
can then be more readily predicted.

Unless otherwise stated within the documentation of a specific resource, this
provider's results are **not** sufficiently random for cryptographic use.

For more information on the specific resources available, see the links in the
navigation bar. Read on for information on the general patterns that apply
to this provider's resources.

## Resource "Keepers"

As noted above, the random resources generate randomness only when they are
created; the results produced are stored in the Terraform state and re-used
until the inputs change, prompting the resource to be recreated.

The resources all provide a map argument called `keepers` that can be populated
with arbitrary key/value pairs that should be selected such that they remain
the same until new random values are desired.

For example:

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

Resource "keepers" are optional. The other arguments to each resource must
*also* remain constant in order to retain a random result.

To force a random result to be replaced, the `taint` command can be used to
produce a new result on the next run.
