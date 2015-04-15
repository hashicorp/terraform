---
layout: "docs"
page_title: "Configuring Atlas"
sidebar_current: "docs-config-atlas"
description: |-
  Atlas is the ideal way to use Terraform in a team environment. Atlas will run Terraform for you, safely handle parallelization across different team members, save run history along with plans, and more.
---

# Atlas Configuration

Terraform can be configured to be able to upload to HashiCorp's
[Atlas](https://atlas.hashicorp.com). This configuration doesn't change
the behavior of Terraform itself, it only configures your Terraform
configuration to support being uploaded to Atlas via the
[push command](/docs/commands/push.html).

For more information on the benefits of uploading your Terraform
configuration to Atlas, please see the
[push command documentation](/docs/commands/push.html).

This page assumes you're familiar with the
[configuration syntax](/docs/configuration/syntax.html)
already.

## Example

Atlas configuration looks like the following:

```
atlas {
	name = "mitchellh/production-example"
}
```

## Description

The `atlas` block configures the settings when Terraform is
[pushed](/docs/commands/push.html) to Atlas. Only one `atlas` block
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

```
atlas {
	name = VALUE
}
```
