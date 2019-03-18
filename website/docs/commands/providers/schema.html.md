---
layout: "commands-providers"
page_title: "Command: providers schema"
sidebar_current: "docs-commands-providers-schema"
description: |-
  The `terraform providers schema` command prints detailed schemas for the providers used
  in the current configuration.
---

# Command: terraform providers schema

The `terraform providers schema` command is used to print detailed schemas for the providers used in the current configuration.

-> `terraform providers schema` requires **Terraform v0.12 or later**.

## Usage

Usage: `terraform providers schema [options]`

The list of available flags are:

* `-json` - Displays the schemas in a machine-readble, JSON format.

Please note that, at this time, the `-json` flag is a _required_ option. In future releases, this command will be extended to allow for additional options. 
