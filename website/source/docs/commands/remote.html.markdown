---
layout: "docs"
page_title: "Command: remote"
sidebar_current: "docs-commands-remote"
description: |-
  The `terraform remote` command is used to configure Terraform to make
  use of remote state storage, change remote storage configuration, or
  to disable it.
---

# Command: remote

The `terraform remote` command is used to configure all aspects of
remote state storage. When remote state storage is enabled,
Terraform will automatically fetch the latest state from the remote
server when necessary and if any updates are made, the newest state
is persisted back to the remote server.
In this mode, users do not need to durably store the state using version
control or shared storage.

## Usage

Usage: `terraform remote SUBCOMMAND [options]`

The `remote` command behaves as another command that further has more
subcommands. The subcommands available are:

  * [config](/docs/commands/remote-config.html) - Configure the remote storage,
      including enabling/disabling it.
  * [pull](/docs/commands/remote-pull.html) - Sync the remote storage to
      the local storage (download).
  * [push](/docs/commands/remote-push.html) - Sync the local storage to
      remote storage (upload).
