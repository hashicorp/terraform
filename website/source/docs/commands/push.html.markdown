---
layout: "docs"
page_title: "Command: push"
sidebar_current: "docs-commands-push"
description: |-
  The `terraform push` command is used to push a cached local copy
  of the state to a remote storage server.
---

# Command: push

The `terraform push` uploads the cached state file to the
remote server when remote state storage is enabled. The [`remote`
command](/docs/commands/remote.html) should be used to enable
remote state storage.

Uploading is typically done automatically when running a Terraform
command that modifies state, but this can be used to retry uploads
if a transient failure occurs.

## Usage

Usage: `terraform push`

The `push` command is invoked without options to upload the
local cached state to the remote storage server.

