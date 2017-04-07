---
layout: "enterprise"
page_title: "Notifications - Runs - Terraform Enterprise"
sidebar_current: "docs-enterprise-runs-notifications"
description: |-
  Terraform Enterprise can send notifications to your organization. This post is on how.
---


# Terraform Run Notifications

Terraform Enterprise can send run notifications, the following events are
configurable:

- **Needs Confirmation** - The plan phase has succeeded, and there are changes
  that need to be confirmed before applying.

- **Confirmed** - A plan has been confirmed, and it will begin applying shortly.

- **Discarded** - A user has discarded the plan.

- **Applying** - The plan has begun to apply and make changes to your
  infrastructure.

- **Applied** - The plan was applied successfully.

- **Errored** - An error has occurred during the plan or apply phase.

> Emails will include logs for the **Needs Confirmation**, **Applied**, and
> **Errored** events.

You can toggle notifications for each of these events on the "Integrations" tab
of an environment.
