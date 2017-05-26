---
layout: "enterprise"
page_title: "Build Notifications - Packer Builds - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerbuilds-notifications"
description: |-
  Terraform Enterprise can send build notifications to your organization.
---

# About Packer Build Notifications

Terraform Enterprise can send build notifications to your organization for the
following events:

- **Starting** - The build has begun.
- **Finished** - All build jobs have finished successfully.
- **Errored** - An error has occurred during one of the build jobs.
- **Canceled** - A user has canceled the build.

> Emails will include logs for the **Finished** and **Errored** events.

You can toggle notifications for each of these events on the "Integrations" tab
of a build configuration.
