---
layout: "commands-state"
page_title: "Command: state pull"
sidebar_current: "docs-state-sub-pull"
description: |-
  The `terraform state pull` command is used to manually download and output the state from remote state.
---

# Command: state pull

The `terraform state pull` command is used to manually download and output
the state from [remote state](/docs/state/remote.html). This command also
works with local state.

## Usage

Usage: `terraform state pull`

This command will download the state from its current location and
output the raw format to stdout.

This is useful for reading values out of state (potentially pairing this
command with something like [jq](https://stedolan.github.io/jq/)). It is
also useful if you need to make manual modifications to state.
