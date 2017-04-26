---
layout: "enterprise"
page_title: "Execution - Runs - Terraform Enterprise"
sidebar_current: "docs-enterprise-runs-execute"
description: |-
  How runs execute in Terraform Enterprise.
---

# How Terraform Runs Execute

This briefly covers the internal process of running Terraform plan and applies.
It is not necessary to know this information, but may be valuable to help
understand implications of running or debugging failed runs.

## Steps of Execution

1. A set of Terraform configuration and directory of files is uploaded via Terraform Push or GitHub
2. Terraform Enterprise creates a version of the Terraform configuration and waits for the upload
to complete. At this point, the version will be visible in the UI even if the upload has
not completed
3. Once the upload finishes, Terraform Enterprise creates a run and queues a `terraform plan`
4. In the run environment, the package including the files and Terraform
configuration are downloaded
5. `terraform plan` is run against the configuration in the run environment
6. Logs are streamed into the UI and stored
7. The `.tfplan` file created in the plan is uploaded and stored
8. Once the plan completes, the environment is torn down and status is
updated in the UI
9. The plan then requires confirmation by an operator. It can optionally
be discarded and ignored at this stage
10. Once confirmed, the run then executes a `terraform apply` in a new
environment against the saved `.tfplan` file
11. The logs are streamed into the UI and stored
12. Once the apply completes, the environment is torn down, status is
updated in the UI and changed state is saved back

Note: In the case of a failed apply, it's safe to re-run. This is possible
because Terraform saves partial state and can "pick up where it left off".

### Customizing Terraform Execution

As described in the steps above, Terraform will be run against your configuration
when changes are pushed via GitHub, `terraform push`, or manually queued in the
UI. There are a few options available to customize the execution of Terraform.
These are:

- The directory that contains your environment's Terraform configuration can be customized
to support directory structures with more than one set of Terraform configuration files.
To customize the directory for your Environment, set the _Terraform Directory_
property in the [_GitHub Integration_](/docs/enterprise/vcs/github.html) settings for your environment. This is equivalent to
passing the `[dir]` argument when running Terraform in your local shell.
- The directory in which Terraform is executed from can be customized to support directory
structures with nested sub-directories or configurations that use Terraform modules with
relative paths. To customize the directory used for Terraform execution in your Environment, set the `TF_ATLAS_DIR`
[environment variable](/docs/enterprise/runs/variables-and-configuration.html#environment-variables)
to the relative path of the directory - ie. `terraform/production`. This is equivalent to
changing directories to the appropriate path in your local shell and then executing Terraform.
