## 1.2.0 (Unreleased)

NEW FEATURES:

* `precondition` and `postcondition` check blocks for resources, data sources, and module output values: module authors can now document assumptions and assertions about configuration and state values. If these conditions are not met, Terraform will report a custom error message to the user and halt further evaluation.
* Terraform now supports [run tasks](https://www.terraform.io/cloud-docs/workspaces/settings/run-tasks), a Terraform Cloud integration for executing remote operations, for the post plan stage of a run.

ENHANCEMENTS:

* The "Invalid for_each argument" error message for unknown maps/sets now includes an additional paragraph to try to help the user notice they can move apply-time values into the map _values_ instead of the map _keys_, and thus avoid the problem without resorting to `-target`. ([#30327](https://github.com/hashicorp/terraform/issues/30327))
* When showing the progress of a remote operation running in Terraform Cloud, Terraform CLI will include information about post-plan [run tasks](https://www.terraform.io/cloud-docs/workspaces/settings/run-tasks). ([#30141](https://github.com/hashicorp/terraform/issues/30141))
* Error messages for preconditions, postconditions, and custom variable validations are now evaluated as expressions, allowing interpolation of relevant values into the output. ([#30613](https://github.com/hashicorp/terraform/issues/30613))
* There are some small improvements to the error and warning messages Terraform will emit in the case of invalid provider configuration passing between modules. There are no changes to which situations will produce errors and warnings, but the messages now include additional information intended to clarify what problem Terraform is describing and how to address it. ([#30639](https://github.com/hashicorp/terraform/issues/30639))
* When running `terraform plan`, only show external changes which may have contributed to the current plan ([#30486](https://github.com/hashicorp/terraform/issues/30486))
* Add `TF_ORGANIZATION` environment variable fallback for `organization` in the cloud configuration
* Add `TF_HOSTNAME` environment variable fallback for `hostname` in the cloud configuration 

BUG FIXES:

* Terraform now handles type constraints, nullability, and custom variable validation properly for root module variables. Previously there was an order of operations problem where the nullability and custom variable validation were checked too early, prior to dealing with the type constraints, and thus that logic could potentially "see" an incorrectly-typed value in spite of the type constraint, leading to incorrect errors. ([#29959](https://github.com/hashicorp/terraform/issues/29959))
* `terraform show -json`: JSON plan output now correctly maps aliased providers to their configurations, and includes the full provider source address alongside the short provider name. ([#30138](https://github.com/hashicorp/terraform/issues/30138))
* Terraform now prints a warning when adding an attribute to `ignore_changes` that is managed only by the provider (non-optional computed attribute). ([#30517](https://github.com/hashicorp/terraform/issues/30517))
* Terraform will prioritize local terraform variables over remote terraform variables in operations such as `import`, `plan`, `refresh` and `apply` for workspaces in local execution mode. This behavior applies to both `remote` backend and the `cloud` integration configuration. ([#29972](https://github.com/hashicorp/terraform/issues/29972))
* Terraform now outputs an error when `cidrnetmask()` is called with an IPv6 address. ([#30703](https://github.com/hashicorp/terraform/issues/30703))

UPGRADE NOTES:

* The Terraform Cloud integration relies on the Go-TFE SDK. Terraform has upgraded this dependency to use its new major version 1.0 [[#30626](https://github.com/hashicorp/terraform/issues/30626)]. [Go-TFE v1.0.0 CHANGELOG](https://github.com/hashicorp/go-tfe/releases/tag/v1.0.0).

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
