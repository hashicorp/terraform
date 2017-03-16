---
title: "Automatic Terraform Applies in Atlas"
---

# Automatic Terraform Applies in Atlas

<div class="alert-infos">
  <div class="alert-info">
    This is an unreleased beta feature. Please <a href="/help/support">contact support</a> if you are interested in helping us test this feature.
  </div>
</div>

Atlas can optionally automatically apply successful Terraform plans to your
infrastructure. This option is disabled by default and can be enabled by an
organization owner on a per-[environment](/help/glossary#environment) basis.

<div class="alert-errors">
  <div class="row alert-error">
    This is an advanced feature that enables changes to active infrastructure
    without user confirmation. Please understand the implications to your
    infrastructure before enabling.
  </div>
</div>

## Enabling Auto-Apply

To enable auto-apply for an environment, visit the environment settings page in
Atlas and check the box labeled "auto apply" and click the save button to
persist the changes. The next successful Terraform plan for the environment will
automatically apply without user confirmation.
