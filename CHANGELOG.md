## 1.2.2 (June 01, 2022)

ENHANCEMENTS:

* Invalid `-var` arguments with spaces between the name and value now have an improved error message ([#30985](https://github.com/hashicorp/terraform/issues/30985))

BUG FIXES:

* Terraform now hides invalid input values for sensitive root module variables when generating error diagnostics ([#30552](https://github.com/hashicorp/terraform/issues/30552))
* Fixed crash on CLI autocomplete ([#31160](https://github.com/hashicorp/terraform/issues/31160))
* The "Configuration contains unknown values" error message now includes attribute paths ([#3111](https://github.com/hashicorp/terraform/issues/3111))

## 1.2.1 (May 23, 2022)

BUG FIXES:

* SSH provisioner connections fail when using signed `ed25519` keys ([#31092](https://github.com/hashicorp/terraform/issues/31092))
* Crash with invalid module source ([#31060](https://github.com/hashicorp/terraform/issues/31060))
* Incorrect "Module is incompatible with count, for_each, and depends_on" error when a provider is nested within a module along with a sub-module using `count` or `for_each` ([#31091](https://github.com/hashicorp/terraform/issues/31091))


## 1.2.0 (May 18, 2022)

UPGRADE NOTES:

* If you use the [third-party credentials helper plugin terraform-credentials-env](https://github.com/apparentlymart/terraform-credentials-env), you should disable it as part of upgrading to Terraform v1.2 because similar functionality is now built in to Terraform itself.

    The new behavior supports the same environment variable naming scheme but has a difference in priority order from the credentials helper: `TF_TOKEN_...` environment variables will now take priority over credentials blocks in CLI configuration and credentials stored automatically by terraform login, which is not true for credentials provided by any credentials helper plugin. If you see Terraform using different credentials after upgrading, check to make sure you do not specify credentials for the same host in multiple locations.

    If you use the credentials helper in conjunction with the [hashicorp/tfe](https://registry.terraform.io/providers/hashicorp/tfe) Terraform provider to manage Terraform Cloud or Terraform Enterprise objects with Terraform, you should also upgrade to version 0.31 of that provider, which added the corresponding built-in support for these environment variables.
* The official Linux packages for the v1.2 series now require Linux kernel version 2.6.32 or later.
* When making outgoing HTTPS or other TLS connections as a client, Terraform now requires the server to support TLS v1.2. TLS v1.0 and v1.1 are no longer supported. Any safely up-to-date server should support TLS 1.2, and mainstream web browsers have required it since 2020.
* When making outgoing HTTPS or other TLS connections as a client, Terraform will no longer accept CA certificates signed using the SHA-1 hash function. Publicly trusted Certificate Authorities have not issued SHA-1 certificates since 2015.

(Note: the changes to Terraform's requirements when interacting with TLS servers apply only to requests made by Terraform CLI itself, such as provider/module installation and state storage requests. Terraform provider plugins include their own TLS clients which may have different requirements, and may add new requirements in their own releases, independently of Terraform CLI changes.)

NEW FEATURES:

* `precondition` and `postcondition` check blocks for resources, data sources, and module output values: module authors can now document assumptions and assertions about configuration and state values. If these conditions are not met, Terraform will report a custom error message to the user and halt further execution.
* `replace_triggered_by` is a new `lifecycle` argument for managed resources which triggers replacement of an object based on changes to an upstream dependency.
* You can now specify credentials for [Terraform-native services](https://www.terraform.io/internals/remote-service-discovery) using an environment variable named as `TF_TOKEN_` followed by an encoded version of the hostname. For example, Terraform will use variable `TF_TOKEN_app_terraform_io` as a bearer token for requests to "app.terraform.io", for the Terraform Cloud integration and private registry requests.

ENHANCEMENTS:

* When showing a plan, Terraform CLI will now only show "Changes outside of Terraform" if they relate to resources and resource attributes that contributed to the changes Terraform is proposing to make. ([#30486](https://github.com/hashicorp/terraform/issues/30486))
* Error messages for preconditions, postconditions, and custom variable validations are now evaluated as expressions, allowing interpolation of relevant values into the output. ([#30613](https://github.com/hashicorp/terraform/issues/30613))
* When showing the progress of a remote operation running in Terraform Cloud, Terraform CLI will include information about post-plan [run tasks](https://www.terraform.io/cloud-docs/workspaces/settings/run-tasks). ([#30141](https://github.com/hashicorp/terraform/issues/30141))
* Terraform will now show a slightly different note in the plan output if a data resource read is deferred to the apply step due to it depending on a managed resource that has changes pending. ([#30971](https://github.com/hashicorp/terraform/issues/30971))
* The "Invalid for_each argument" error message for unknown maps/sets now includes an additional paragraph to try to help the user notice they can move apply-time values into the map _values_ instead of the map _keys_, and thus avoid the problem without resorting to `-target`. ([#30327](https://github.com/hashicorp/terraform/issues/30327))
* There are some small improvements to the error and warning messages Terraform will emit in the case of invalid provider configuration passing between modules. There are no changes to which situations will produce errors and warnings, but the messages now include additional information intended to clarify what problem Terraform is describing and how to address it. ([#30639](https://github.com/hashicorp/terraform/issues/30639))
* The environment variables `TF_CLOUD_ORGANIZATION` and `TF_CLOUD_HOSTNAME` now serve as fallbacks for the arguments of the same name inside a `cloud` block configuring integration with Terraform Cloud.
* The environment variable `TF_WORKSPACE` will now additionally serve as an implicit configuration of a single selected workspace on Terraform Cloud if (and only if) the `cloud` block does not include an explicit workspaces configuration.
* The AzureRM Backend now defaults to using MSAL (and Microsoft Graph) rather than ADAL (and Azure Active Directory Graph) for authentication. ([#30891](https://github.com/hashicorp/terraform/issues/30891))
* The AzureRM Backend now supports authenticating as a service principal using OpenID Connect. ([#30936](https://github.com/hashicorp/terraform/pull/30936))
* When running on macOS, Terraform will now use platform APIs to validate certificates presented by TLS (HTTPS) servers. This may change exactly which root certificates Terraform will accept as valid. ([#30768](https://github.com/hashicorp/terraform/issues/30768))
* Show remote host in error message for clarity when installation of provider fails ([#30810](https://github.com/hashicorp/terraform/issues/30810))
* Terraform now prints a warning when adding an attribute to `ignore_changes` that is managed only by the provider. Specifying non-configurable attributes in `ignore_changes` has no effect because `ignore_changes` tells Terraform to ignore future changes made in the configuration. ([#30517](https://github.com/hashicorp/terraform/issues/30517))
* `terraform show -json` now includes exact type information for output values. ([#30945](https://github.com/hashicorp/terraform/issues/30945))
* The `ssh` provisioner connection now supports SSH over HTTP proxy. ([#30274](https://github.com/hashicorp/terraform/pull/30274))
* * The SSH client for provisioners now supports newer key algorithms, allowing it to connect to servers running more recent versions of OpenSSH. ([#30962](https://github.com/hashicorp/terraform/issues/30962))

BUG FIXES:

* Terraform now handles type constraints, nullability, and custom variable validation properly for root module variables. Previously there was an order of operations problem where the nullability and custom variable validation were checked too early, prior to dealing with the type constraints, and thus that logic could potentially "see" an incorrectly-typed value in spite of the type constraint, leading to incorrect errors. ([#29959](https://github.com/hashicorp/terraform/issues/29959))
* When reporting a type mismatch between the true and false results of a conditional expression when both results are of the same structural type kind (object/tuple, or a collection thereof), Terraform will no longer return a confusing message like "the types are object and object, respectively", and will instead attempt to explain how the two structural types differ. ([#30920](https://github.com/hashicorp/terraform/issues/30920))
* Applying the various type conversion functions like `tostring`, `tonumber`, etc to `null` will now return a null value of the intended type. For example, `tostring(null)` converts from a null value of an unknown type to a null value of string type. Terraform can often handle such conversions automatically when needed, but explicit annotations like this can help Terraform to understand author intent when inferring type conversions for complex-typed values. ([#30879](https://github.com/hashicorp/terraform/issues/30879))
* Terraform now returns an error when `cidrnetmask()` is called with an IPv6 address, as it was previously documented to do. IPv6 standards do not preserve the "netmask" syntax sometimes used for IPv4 network configuration; use CIDR prefix syntax instead. ([#30703](https://github.com/hashicorp/terraform/issues/30703))
* When performing advanced state management with the `terraform state` commands, Terraform now checks the `required_version` field in the configuration before proceeding. ([#30511](https://github.com/hashicorp/terraform/pull/30511))
* When rendering a diff, Terraform now quotes the name of any object attribute whose string representation is not a valid identifier. ([#30766](https://github.com/hashicorp/terraform/issues/30766))
* Terraform will now prioritize local terraform variables over remote terraform variables in operations such as `import`, `plan`, `refresh` and `apply` for workspaces in local execution mode. This behavior applies to both `remote` backend and the `cloud` integration configuration. ([#29972](https://github.com/hashicorp/terraform/issues/29972))
* `terraform show -json`: JSON plan output now correctly maps aliased providers to their configurations, and includes the full provider source address alongside the short provider name. ([#30138](https://github.com/hashicorp/terraform/issues/30138))
* The local token configuration in the `cloud` and `remote` backend now has higher priority than a token specified in a `credentials` block in the CLI configuration. ([#30664](https://github.com/hashicorp/terraform/issues/30664)) 
* The `cloud` integration now gracefully exits when `-input=false` and an operation requires some user input.
* Terraform will now reliably detect an interruption (e.g. Ctrl+C) during planning for `terraform apply -auto-approve`. Previously there was a window of time where interruption would cancel the plan step but not prevent Terraform from proceeding to the apply step. ([#30979](https://github.com/hashicorp/terraform/issues/30979))
* Terraform will no longer crash if a provider fails to return a schema. ([#30987](https://github.com/hashicorp/terraform/issues/30987))

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
