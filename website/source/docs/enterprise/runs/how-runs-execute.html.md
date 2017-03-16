---
title: "How Terraform Runs Execute in Atlas"
---

# How Terraform Runs Execute in Atlas

This briefly covers the internal process of running Terraform plan and
applies in Atlas. It is not necessary to know this information, but may be
valuable to help understand implications of running in Atlas or debug failing
runs.

## Steps of Execution

1. A set of Terraform configuration and directory of files is uploaded via Terraform Push or GitHub
1. Atlas creates a version of the Terraform configuration and waits for the upload
to complete. At this point, the version will be visible in the UI even if the upload has
not completed
1. Once the upload finishes, Atlas creates a run and queues a `terraform plan`
1. In the run environment, the package including the files and Terraform
configuration are downloaded
1. `terraform plan` is run against the configuration in the run environment
1. Logs are streamed into the UI and stored
1. The `.tfplan` file created in the plan is uploaded and stored
1. Once the plan completes, the environment is torn down and status is
updated in the UI
1. The plan then requires confirmation by an operator. It can optionally
be discarded and ignored at this stage
1. Once confirmed, the run then executes a `terraform apply` in a new
environment against the saved `.tfplan` file
1. The logs are streamed into the UI and stored
1. Once the apply completes, the environment is torn down, status is
updated in the UI and changed state is saved back to Atlas

Note: In the case of a failed apply, it's safe to re-run. This is possible
because Terraform saves partial state and can "pick up where it left off".

### Customizing Terraform Execution

As described in the steps above, Atlas will run Terraform against your configuration
when changes are pushed via GitHub, `terraform push`, or manually queued in the 
Atlas UI. There are a few options available to customize the execution of Terraform.
These are:

- The directory that contains your environment's Terraform configuration can be customized 
to support directory structures with more than one set of Terraform configuration files.
To customize the directory for your Atlas Environment, set the _Terraform Directory_ 
property in the _GitHub Integration_ settings for your environment. This is equivalent to 
passing the `[dir]` argument when running Terraform in your local shell.
- The directory in which Terraform is executed from can be customized to support directory 
structures with nested sub-directories or configurations that use Terraform modules with 
relative paths. To customize the directory used for Terraform execution in your Atlas 
Environment, set the `TF_ATLAS_DIR` 
[environment variable](/help/terraform/runs/variables-and-configuration#environment-variables)
to the relative path of the directory - ie. `terraform/production`. This is equivalent to 
changing directories to the appropriate path in your local shell and then executing Terraform.
