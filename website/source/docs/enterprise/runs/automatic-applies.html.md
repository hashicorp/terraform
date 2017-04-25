---
layout: "enterprise"
page_title: "Automatic Applies - Runs - Terraform Enterprise"
sidebar_current: "docs-enterprise-runs-applies"
description: |-
  How to automatically apply plans.
---

# Automatic Terraform Applies

-> This is an unreleased beta feature. Please
<a href="mailto:support@hashicorp.com">contact support</a> if you are interested
in helping us test this feature.

You can automatically apply successful Terraform plans to your
infrastructure. This option is disabled by default and can be enabled by an
organization owner on a per-environment basis.

-> This is an advanced feature that enables changes to active infrastructure
without user confirmation. Please understand the implications to your
infrastructure before enabling.

## Enabling Auto-Apply

To enable auto-apply for an environment, visit the environment settings page
check the box labeled "auto apply" and click the save button to persist the
changes. The next successful Terraform plan for the environment will
automatically apply without user confirmation.
