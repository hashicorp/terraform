## 1.2.0 (Unreleased)

UPGRADE NOTES:

* The official Linux packages for the v1.2 series now require Linux kernel version 2.6.32 or later.
* When making outgoing HTTPS or other TLS connections as a client, Terraform now requires the server to support TLS v1.2. TLS v1.0 and v1.1 are no longer supported. Any safely up-to-date server should support TLS 1.2, and mainstream web browsers have required it since 2020.
* When making outgoing HTTPS or other TLS connections as a client, Terraform will no longer accept CA certificates signed using the SHA-1 hash function. Publicly trusted Certificate Authorities have not issued SHA-1 certificates since 2015.

    (Note: the changes to Terraform's requirements when interacting with TLS servers apply only to requests made by Terraform CLI itself, such as provider/module installation and state storage requests. Terraform provider plugins include their own TLS clients which may have different requirements, and may add new requirements in their own releases, independently of Terraform CLI changes.)
* If you use the [third-party credentials helper plugin terraform-credentials-env](https://github.com/apparentlymart/terraform-credentials-env), you should disable it as part of upgrading to Terraform v1.2 because similar functionality is now built in to Terraform itself.

    The new behavior supports the same environment variable naming scheme but has a difference in priority order from the credentials helper: `TF_TOKEN_...` environment variables will now take priority over credentials blocks in CLI configuration and credentials stored automatically by terraform login, which is not true for credentials provided by any credentials helper plugin. If you see Terraform using different credentials after upgrading, check to make sure you do not specify credentials for the same host in multiple locations.

    If you use the credentials helper in conjunction with the [hashicorp/tfe](https://registry.terraform.io/providers/hashicorp/tfe) Terraform provider to manage Terraform Cloud or Terraform Enterprise objects with Terraform, you should also upgrade to version 0.31 of that provider, which added the corresponding built-in support for these environment variables.

NEW FEATURES:

* `precondition` and `postcondition` check blocks for resources, data sources, and module output values: module authors can now document assumptions and assertions about configuration and state values. If these conditions are not met, Terraform will report a custom error message to the user and halt further evaluation.
* You may specify remote network service credentials using an environment variable named after the host name with a `TF_TOKEN_` prefix. For example, the value of a variable named `TF_TOKEN_app_terraform_io` will be used as a bearer authorization token when the CLI makes service requests to the host name "app.terraform.io".
* `replace_triggered_by` is a new `lifecycle` argument which allows one to configure the replacement of a resource based on changes in a dependency.

ENHANCEMENTS:

* The "Invalid for_each argument" error message for unknown maps/sets now includes an additional paragraph to try to help the user notice they can move apply-time values into the map _values_ instead of the map _keys_, and thus avoid the problem without resorting to `-target`. ([#30327](https://github.com/hashicorp/terraform/issues/30327))
* When showing the progress of a remote operation running in Terraform Cloud, Terraform CLI will include information about post-plan [run tasks](https://www.terraform.io/cloud-docs/workspaces/settings/run-tasks). ([#30141](https://github.com/hashicorp/terraform/issues/30141))
* Error messages for preconditions, postconditions, and custom variable validations are now evaluated as expressions, allowing interpolation of relevant values into the output. ([#30613](https://github.com/hashicorp/terraform/issues/30613))
* There are some small improvements to the error and warning messages Terraform will emit in the case of invalid provider configuration passing between modules. There are no changes to which situations will produce errors and warnings, but the messages now include additional information intended to clarify what problem Terraform is describing and how to address it. ([#30639](https://github.com/hashicorp/terraform/issues/30639))
* When running `terraform plan`, only show external changes which may have contributed to the current plan ([#30486](https://github.com/hashicorp/terraform/issues/30486))
* Add `TF_CLOUD_ORGANIZATION` environment variable fallback for `organization` in the cloud configuration
* Add `TF_CLOUD_HOSTNAME` environment variable fallback for `hostname` in the cloud configuration
* `TF_WORKSPACE` can now be used to configure the `workspaces` attribute in your cloud configuration
* When running on macOS, Terraform will now use platform APIs to validate certificates presented by TLS (HTTPS) servers. This may change exactly which root certificates Terraform will accept as valid. ([#30768](https://github.com/hashicorp/terraform/issues/30768))
* The AzureRM Backend now defaults to using MSAL (and Microsoft Graph) rather than ADAL (and Azure Active Directory Graph) for authentication. ([#30891](https://github.com/hashicorp/terraform/issues/30891))
* Show remote host in error message for clarity when installation of provider fails ([#30810](https://github.com/hashicorp/terraform/issues/30810))
* Terraform now prints a warning when adding an attribute to `ignore_changes` that is managed only by the provider (non-optional computed attribute). ([#30517](https://github.com/hashicorp/terraform/issues/30517))

BUG FIXES:

* Terraform now handles type constraints, nullability, and custom variable validation properly for root module variables. Previously there was an order of operations problem where the nullability and custom variable validation were checked too early, prior to dealing with the type constraints, and thus that logic could potentially "see" an incorrectly-typed value in spite of the type constraint, leading to incorrect errors. ([#29959](https://github.com/hashicorp/terraform/issues/29959))
* Applying the various type conversion functions like `tostring`, `tonumber`, etc to `null` will now return a null value of the intended type. For example, `tostring(null)` converts from a null value of an unknown type to a null value of string type. Terraform can often handle such conversions automatically when needed, but explicit annotations like this can help Terraform to understand author intent when inferring type conversions for complex-typed values. [GH-30879]
* Terraform now outputs an error when `cidrnetmask()` is called with an IPv6 address, as it was previously documented to do. ([#30703](https://github.com/hashicorp/terraform/issues/30703))
* When performing advanced state management with the `terraform state` commands, Terraform now checks the `required_version` field in the configuration before proceeding. ([#30511](https://github.com/hashicorp/terraform/pull/30511))
* When rendering a diff, Terraform now quotes the name of any object attribute whose string representation is not a valid identifier. ([#30766](https://github.com/hashicorp/terraform/issues/30766))
* Terraform will prioritize local terraform variables over remote terraform variables in operations such as `import`, `plan`, `refresh` and `apply` for workspaces in local execution mode. This behavior applies to both `remote` backend and the `cloud` integration configuration. ([#29972](https://github.com/hashicorp/terraform/issues/29972))
* `terraform show -json`: JSON plan output now correctly maps aliased providers to their configurations, and includes the full provider source address alongside the short provider name. ([#30138](https://github.com/hashicorp/terraform/issues/30138))

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
