---
layout: "docs"
page_title: "Terraform Push - 0.11 Configuration Language"
sidebar_current: "docs-conf-old-push"
description: |-
  Terraform's push command was a way to interact with the legacy version of Terraform Enterprise. It is not supported in the current version of Terraform Enterprise.
---

# Terraform Push Configuration

-> **Note:** This page is about Terraform 0.11 and earlier, and documents a
feature that was removed in Terraform 0.12.

~> **Important:** The `terraform push` command is deprecated, and only works with the legacy version of Terraform Enterprise. In the current version of Terraform Cloud, you can upload configurations using the API. See [the docs about API-driven runs](/docs/cloud/run/api.html) for more details.

The [`terraform push` command](/docs/commands/push.html) uploads a configuration to a Terraform Enterprise (legacy) environment. The name of the environment (and the organization it's in) can be specified on the command line, or as part of the Terraform configuration in an `atlas` block.

The `atlas` block does not configure remote state; it only configures the push command. For remote state, [use a `terraform { backend "<NAME>" {...} }` block](/docs/backends/config.html).

This page assumes you're familiar with the
[configuration syntax](./syntax.html)
already.

## Example

Terraform push configuration looks like the following:

```hcl
atlas {
  name = "mitchellh/production-example"
}
```

~> **Why is this called "atlas"?** Atlas was previously a commercial offering
from HashiCorp that included a full suite of enterprise products. The products
have since been broken apart into their individual products, like **Terraform
Enterprise**. While this transition is in progress, you may see references to
"atlas" in the documentation. We apologize for the inconvenience.

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
