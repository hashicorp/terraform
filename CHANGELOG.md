## 1.2.0 (Unreleased)

NEW FEATURES:

* `precondition` and `postcondition` check blocks for resources, data sources, and module output values: module authors can now document assumptions and assertions about configuration and state values. If these conditions are not met, Terraform will report a custom error message to the user and halt further evaluation.
* Terraform now supports [run tasks](https://www.terraform.io/cloud-docs/workspaces/settings/run-tasks), a Terraform Cloud integration for executing remote operations, for the post plan stage of a run.

ENHANCEMENTS:

* The "Invalid for_each argument" error message for unknown maps/sets now includes an additional paragraph to try to help the user notice they can move apply-time values into the map _values_ instead of the map _keys_, and thus avoid the problem without resorting to `-target`. [GH-30327]
* When showing the progress of a remote operation running in Terraform Cloud, Terraform CLI will include information about post-plan [run tasks](https://www.terraform.io/cloud-docs/workspaces/settings/run-tasks). [GH-30141]
* Error messages for preconditions, postconditions, and custom variable validations are now evaluated as expressions, allowing interpolation of relevant values into the output. [GH-30613]
* There are some small improvements to the error and warning messages Terraform will emit in the case of invalid provider configuration passing between modules. There are no changes to which situations will produce errors and warnings, but the messages now include additional information intended to clarify what problem Terraform is describing and how to address it. [GH-30639]

BUG FIXES:

* Terraform now handles type constraints, nullability, and custom variable validation properly for root module variables. Previously there was an order of operations problem where the nullability and custom variable validation were checked too early, prior to dealing with the type constraints, and thus that logic could potentially "see" an incorrectly-typed value in spite of the type constraint, leading to incorrect errors. [GH-29959]
* `terraform show -json`: JSON plan output now correctly maps aliased providers to their configurations, and includes the full provider source address alongside the short provider name. [GH-30138]
* Terraform now prints a warning when adding an attribute to `ignore_changes` that is managed only by the provider (non-optional computed attribute). [GH-30517]
* Terraform will prioritize local terraform variables over remote terraform variables in operations such as `import`, `plan`, `refresh` and `apply` for workspaces in local execution mode. This behavior applies to both `remote` backend and the `cloud` integration configuration. [GH-29972]


## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
