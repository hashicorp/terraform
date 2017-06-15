---
layout: "docs"
page_title: "Configuring Terraform Enterprise"
sidebar_current: "docs-config-terraform-enterprise"
description: |-
  Terraform Enterprise is the ideal way to use Terraform in a team environment. Terraform Enterprise will run Terraform for you, safely handle parallelization across different team members, save run history along with plans, and more.
---

# Terraform Enterprise Configuration

Terraform can be configured to be able to upload to HashiCorp's
[Terraform Enterprise](https://www.hashicorp.com/products/terraform/). This configuration doesn't change
the behavior of Terraform itself, it only configures your Terraform
configuration to support being uploaded to Terraform Enterprise via the
[push command](/docs/commands/push.html).

For more information on the benefits of uploading your Terraform
configuration to Terraform Enterprise, please see the
[push command documentation](/docs/commands/push.html).

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

~> **Why is this called "atlas"?** Atlas was previously a commercial offering
from HashiCorp that included a full suite of enterprise products. The products
have since been broken apart into their individual products, like **Terraform
Enterprise**. While this transition is in progress, you may see references to
"atlas" in the documentation. We apologize for the inconvenience.

## Example

Terraform Enterprise configuration looks like the following:

```hcl
atlas {
  name = "mitchellh/production-example"
}
```

## Description

The `atlas` block configures the settings when Terraform is
[pushed](/docs/commands/push.html) to Terraform Enterprise. Only one `atlas` block
is allowed.

Within the block (the `{ }`) is configuration for Atlas uploading.
No keys are required, but the key typically set is `name`.

**No value within the `atlas` block can use interpolations.** Due
to the nature of this configuration, interpolations are not possible.
If you want to parameterize these settings, use the Atlas block to
set defaults, then use the command-line flags of the
[push command](/docs/commands/push.html) to override.

## Syntax

The full syntax is:

```text
atlas {
  name = VALUE
}
```
