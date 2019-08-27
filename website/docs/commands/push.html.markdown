---
layout: "docs"
page_title: "Command: push"
sidebar_current: "docs-commands-push"
description: |-
  DISCONTINUED. Terraform Cloud and the modern "remote" backend have replaced the old `terraform push` command.
---

# Command: push

!> **Important:** The `terraform push` command is no longer functional. Its functionality was replaced and surpassed by [the `remote` backend](/docs/backends/types/remote.html), which works with current versions of Terraform Cloud. The `remote` backend allows you to run remote operations directly from the command line, and displays real-time output from the remote run environment.

The `terraform push` command was an early implementation of remote Terraform runs. It allowed teams to push a configuration to a remote run environment in a discontinued version of Terraform Enterprise.

The legacy Terraform Enterprise version that supported `terraform push` is no longer available, and there are no remaining instances of that version in operation. Without a service to push to, the command is now completely non-functional.

