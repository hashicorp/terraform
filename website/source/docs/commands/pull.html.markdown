---
layout: "docs"
page_title: "Command: pull"
sidebar_current: "docs-commands-pull"
description: |-
  The `terraform pull` refreshes the cached state file from the
  remote server when remote state storage is enabled.
---

# Command: pull

The `terraform pull` refreshes the cached state file from the
remote server when remote state storage is enabled. The [`remote`
command](/docs/commands/remote.html) should be used to enable
remote state storage.

## Usage

Usage: `terraform pull`

The `pull` command is invoked without options to refresh the
cache copy of the state.

