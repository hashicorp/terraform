---
layout: "docs"
page_title: "Command: remote pull"
sidebar_current: "docs-commands-remote-pull"
description: |-
  The `terraform remote pull` refreshes the cached state file from the
  remote server when remote state storage is enabled.
---

# Command: remote pull

The `terraform remote pull` refreshes the cached state file from the
remote server when remote state storage is enabled. The [`remote config`
command](/docs/commands/remote-config.html) should be used to enable
remote state storage.

## Usage

Usage: `terraform remote pull`

The `remote pull` command is invoked without options to refresh the
cache copy of the state.

