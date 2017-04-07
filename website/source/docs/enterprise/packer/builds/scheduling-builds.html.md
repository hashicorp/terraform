---
layout: "enterprise"
page_title: "Schedule Periodic Builds - Packer Builds - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerbuilds-scheduling"
description: |-
  Terraform Enterprise can automatically run a Packer build and create artifacts on a specified schedule.
---

# Schedule Periodic Builds in Terraform Enterprise

Terraform Enterprise can automatically run a Packer build and
create artifacts on a specified schedule. This option is disabled by default and can be enabled by an
organization owner on a per-[environment](/docs/enterprise/glossary#environment) basis.

On the specified interval, builds will be automatically queued that run Packer
for you, creating any artifacts and sending the appropriate notifications.

If your artifacts are used in any other environments and you have activated the
plan on artifact upload feature, this may also queue Terraform plans.

This feature is useful for maintenance of images and automatic updates, or to
build nightly style images for staging or development environments.

## Enabling Periodic Builds

To enable periodic builds for a build, visit the build settings page and select
the desired interval and click the save button to persist the changes. An
initial build may immediately run, depending on the history, and then will
automatically build at the specified interval.

If you have run a build separately, either manually or triggered from GitHub or
Packer configuration version uploads, Terraform Enterprise will not queue a new
build until the allowed time after the manual build ran. This ensures that a
build has been executed at the specified schedule.
