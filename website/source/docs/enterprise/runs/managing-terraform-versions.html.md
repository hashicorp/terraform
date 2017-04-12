---
layout: "enterprise"
page_title: "Managing Terraform Versions - Runs - Terraform Enterprise"
sidebar_current: "docs-enterprise-runs-versions"
description: |-
  How to manage versions of Terraform Enterprise.
---

# Managing Terraform Versions

Terraform Enterprise does not automatically upgrade the version of Terraform
used to execute plans and applies. This is intentional, as occasionally there
can be backwards incompatible changes made to Terraform that cause state and
plans to differ based on the same configuration, or new versions that produce
some other unexpected behavior.

All upgrades must be performed by a user, but Terraform Enterprise will display
a notice above any plans or applies run with out of date versions. We encourage
the use of the latest version when possible.

Note that regardless of when an upgrade is performed, the version of Terraform
used in a plan will be used in the subsequent apply.

### Upgrading Terraform

1. Go the Settings tab of an environment
2. Go to the "Terraform Version" section and select the version you
wish to use
3. Review the changelog for that version and previous versions
4. Click the save button. At this point, future builds will use that
version
