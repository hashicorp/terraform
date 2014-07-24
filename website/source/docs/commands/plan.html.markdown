---
layout: "docs"
page_title: "Command: plan"
sidebar_current: "docs-commands-plan"
---

# Command: plan

The `terraform plan` command is used to create an execution plan. Terraform
performs a refresh, unless explicitly disabled, and then determines what
actions are necessary to achieve the desired state specified in the
configuration files. The plan can be saved using `-out`, and then provided
to `terraform apply` to ensure only the pre-planned actions are executed.

## Usage

Usage: `terraform plan [options] [dir]`

By default, `plan` requires no flags and looks in the current directory
for the configuration and state file to refresh.

The command-line flags are all optional. The list of available flags are:

* `-destroy` - If set, generates a plan to destroy all the known resources.

* `-no-color` - Disables output with coloring.

* `-out=path` - The path to save the generated execution plan.

* `-refresh=true` - Update the state prior to checking for differences.

* `-state=path` - Path to the state file. Defaults to "terraform.tfstate".

* `-var 'foo=bar'` - Set a variable in the Terraform configuration. This
  flag can be set multiple times.

* `-var-file=foo` - Set variables in the Terraform configuration from
   a file. If "terraform.tfvars" is present, it will be automatically
   loaded if this flag is not specified.

