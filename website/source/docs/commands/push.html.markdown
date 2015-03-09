---
layout: "docs"
page_title: "Command: push"
sidebar_current: "docs-commands-push"
description: |-
  The `terraform push` command is used to upload the Terraform configuration to HashiCorp's Atlas service for automatically managing your infrastructure in the cloud.
---

# Command: push

The `terraform push` command uploads your Terraform configuration to
be managed by HashiCorp's [Atlas](https://atlas.hashicorp.com).
By uploading your configuration to Atlas, Atlas can automatically run
Terraform for you, will save all state transitions, will save plans,
and will keep a history of all Terraform runs.

This makes it significantly easier to use Terraform as a team: team
members modify the Terraform configurations locally and continue to
use normal version control. When the Terraform configurations are ready
to be run, they are pushed to Atlas, and any member of your team can
run Terraform with the push of a button.

Atlas can also be used to set ACLs on who can run Terraform, and a
future update of Atlas will allow parallel Terraform runs and automatically
perform infrastructure locking so only one run is modifying the same
infrastructure at a time.

## Usage

Usage: `terraform push [options] [path]`

The `path` argument is the same as for the
[apply](/docs/commands/apply.html) command.

The command-line flags are all optional. The list of available flags are:

* `-module-upload=true` - If true (default), then the
  [modules](/docs/modules/index.html)
  being used are all locked at their current checkout and uploaded
  completely to Atlas. This prevents Atlas from running `terraform get`
  for you.

* `-name=<name>` - Name of the infrastructure configuration in Atlas.
  The format of this is: "username/name" so that you can upload
  configurations not just to your account but to other accounts and
  organizations. This setting can also be set in the configuration
  in the
  [Atlas section](#).

* `-no-color` - Disables output with coloring

* `-token=<token>` - Atlas API token to use to authorize the upload.
  If blank, the `ATLAS_TOKEN` environmental variable will be used.
