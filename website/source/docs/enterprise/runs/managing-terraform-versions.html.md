---
title: "Managing Terraform Versions"
---

# Managing Terraform Versions

Atlas does not automatically upgrade the version of Terraform
used to execute plans and applies. This is intentional, as occasionally
there can be backwards incompatible changes made to Terraform that cause state
and plans to differ based on the same configuration,
or new versions that produce some other unexpected behavior.

All upgrades must be performed by a user, but Atlas will display a notice
above any plans or applies run with out of date versions. We encourage the use
of the latest version when possible.

Note that regardless of when an upgrade is performed, the version of
Terraform used in a plan will be used in the subsequent apply.

### Upgrading Terraform

1. Go the Settings tab of an environment
1. Go to the "Terraform Version" section and select the version you
wish to use
1. Review the changelog for that version and previous versions
1. Click the save button. At this point, future builds will use that
version
