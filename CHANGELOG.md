## 1.3.7 (January 04, 2023)

BUG FIXES:

* Fix exact version constraint parsing for modules using prerelease versions ([#32377](https://github.com/hashicorp/terraform/issues/32377))
* Prevent panic when a provider returns a null block value during refresh which is used as configuration via `ignore_changes` ([#32428](https://github.com/hashicorp/terraform/issues/32428))

## 1.3.6 (November 30, 2022)

BUG FIXES:

* Terraform could crash if an orphaned resource instance was deleted externally and had condition checks in the configuration ([#32246](https://github.com/hashicorp/terraform/issues/32246))
* Module output changes were being removed and re-added to the stored plan, impacting performance with large numbers of outputs ([#32307](https://github.com/hashicorp/terraform/issues/32307))

## 1.3.5 (November 17, 2022)

BUG FIXES:

* Prevent crash while serializing the plan for an empty destroy operation ([#32207](https://github.com/hashicorp/terraform/issues/32207))
* Allow a destroy plan to refresh instances while taking into account that some may no longer exist ([#32208](https://github.com/hashicorp/terraform/issues/32208))
* Fix Terraform creating objects that should not exist in variables that specify default attributes in optional objects. ([#32178](https://github.com/hashicorp/terraform/issues/32178))
* Fix several Terraform crashes that are caused by HCL creating objects that should not exist in variables that specify default attributes in optional objects within collections. ([#32178](https://github.com/hashicorp/terraform/issues/32178))
* Fix inconsistent behaviour in empty vs null collections. ([#32178](https://github.com/hashicorp/terraform/issues/32178))
* Prevent file uploads from creating unneeded temporary files when the payload size is known ([#32206](https://github.com/hashicorp/terraform/issues/32206))
* Nested attributes marked sensitive by schema no longer reveal sub-attributes in the plan diff ([#32004](https://github.com/hashicorp/terraform/issues/32004))
* Nested attributes now more consistently display when they become unknown or null values in the plan diff ([#32004](https://github.com/hashicorp/terraform/issues/32004))
* Sensitive values are now always displayed as `(sensitive value)` instead of sometimes as `(sensitive)` ([#32004](https://github.com/hashicorp/terraform/issues/32004))


## 1.3.4 (November 02, 2022)

BUG FIXES:

* Fix invalid refresh-only plan caused by data sources being deferred to apply ([#32111](https://github.com/hashicorp/terraform/issues/32111))
* Optimize the handling of condition checks during apply to prevent performance regressions with large numbers of instances ([#32123](https://github.com/hashicorp/terraform/issues/32123))
* Output preconditions should not be evaluated during destroy ([#32051](https://github.com/hashicorp/terraform/issues/32051))
* Fix crash from `console` when outputs contain preconditions ([#32051](https://github.com/hashicorp/terraform/issues/32051))
* Destroy with no state would still attempt to evaluate some values ([#32051](https://github.com/hashicorp/terraform/issues/32051))
* Prevent unnecessary evaluation and planning of resources during the pre-destroy refresh ([#32051](https://github.com/hashicorp/terraform/issues/32051))
* AzureRM Backend: support for generic OIDC authentication via the `oidc_token` and `oidc_token_file_path` properties ([#31966](https://github.com/hashicorp/terraform/issues/31966))
* Input and Module Variables: Convert variable types before attempting to apply default values. ([#32027](https://github.com/hashicorp/terraform/issues/32027))
* When installing remote module packages delivered in tar format, Terraform now limits the tar header block size to 1MiB to avoid unbounded memory usage for maliciously-crafted module packages. ([#32135](https://github.com/hashicorp/terraform/issues/32135))
* Terraform will now reject excessively-complex regular expression patterns passed to the `regex`, `regexall`, and `replace` functions, to avoid unbounded memory usage for maliciously-crafted patterns. This change should not affect any reasonable patterns intended for practical use. ([#32135](https://github.com/hashicorp/terraform/issues/32135))
* Terraform on Windows now rejects invalid environment variables whose values contain the NUL character when propagating environment variables to a child process such as a provider plugin. Previously Terraform would incorrectly treat that character as a separator between two separate environment variables. ([#32135](https://github.com/hashicorp/terraform/issues/32135))

## 1.3.3 (October 19, 2022)

BUG FIXES:

* Fix error when removing a resource from configuration which according to the provider has already been deleted. ([#31850](https://github.com/hashicorp/terraform/issues/31850))
* Fix error when setting empty collections into variables with collections of nested objects with default values. ([#32033](https://github.com/hashicorp/terraform/issues/32033))

## 1.3.2 (October 06, 2022)

BUG FIXES:

* Fixed a crash caused by Terraform incorrectly re-registering output value preconditions during the apply phase (rather than just reusing the already-planned checks from the plan phase). ([#31890](https://github.com/hashicorp/terraform/issues/31890))
* Prevent errors when the provider reports that a deposed instance no longer exists ([#31902](https://github.com/hashicorp/terraform/issues/31902))
* Using `ignore_changes = all` could cause persistent diffs with legacy providers ([#31914](https://github.com/hashicorp/terraform/issues/31914))
* Fix cycles when resource dependencies cross over between independent provider configurations ([#31917](https://github.com/hashicorp/terraform/issues/31917))
* Improve handling of missing resource instances during `import` ([#31878](https://github.com/hashicorp/terraform/issues/31878))

## 1.3.1 (September 28, 2022)

NOTE:
* On `darwin/amd64` and `darwin/arm64` architectures, `terraform` binaries are now built with CGO enabled. This should not have any user-facing impact, except in cases where the pure Go DNS resolver causes problems on recent versions of macOS: using CGO may mitigate these issues. Please see the upstream bug https://github.com/golang/go/issues/52839 for more details.

BUG FIXES:

* Fixed a crash when using objects with optional attributes and default values in collections, most visible with nested modules. ([#31847](https://github.com/hashicorp/terraform/issues/31847))
* Prevent cycles in some situations where a provider depends on resources in the configuration which are participating in planned changes. ([#31857](https://github.com/hashicorp/terraform/issues/31857))
* Fixed an error when attempting to destroy a configuration where resources do not exist in the state. ([#31858](https://github.com/hashicorp/terraform/issues/31858))
* Data sources which cannot be read during will no longer prevent the state from being serialized. ([#31871](https://github.com/hashicorp/terraform/issues/31871))
* Fixed a crash which occured when a resource with a precondition and/or a postcondition appeared inside a module with two or more instances. ([#31860](https://github.com/hashicorp/terraform/issues/31860))

## 1.3.0 (September 21, 2022)

NEW FEATURES:

* **Optional attributes for object type constraints:** When declaring an input variable whose type constraint includes an object type, you can now declare individual attributes as optional, and specify a default value to use if the caller doesn't set it. For example:

    ```terraform
    variable "with_optional_attribute" {
      type = object({
        a = string                # a required attribute
        b = optional(string)      # an optional attribute
        c = optional(number, 127) # an optional attribute with a default value
      })
    }
    ```

    Assigning `{ a = "foo" }` to this variable will result in the value `{ a = "foo", b = null, c = 127 }`.

* Added functions: `startswith` and `endswith` allow you to check whether a given string has a specified prefix or suffix. ([#31220](https://github.com/hashicorp/terraform/issues/31220))

UPGRADE NOTES:

* `terraform show -json`: Output changes now include more detail about the unknown-ness of the planned value. Previously, a planned output would be marked as either fully known or partially unknown, with the `after_unknown` field having value `false` or `true` respectively. Now outputs correctly expose the full structure of unknownness for complex values, allowing consumers of the JSON output format to determine which values in a collection are known only after apply.
* `terraform import`: The `-allow-missing-config` has been removed, and at least an empty configuration block must exist to import a resource.
* Consumers of the JSON output format expecting on the `after_unknown` field to be only `false` or `true` should be updated to support [the change representation](https://www.terraform.io/internals/json-format#change-representation) described in the documentation, and as was already used for resource changes. ([#31235](https://github.com/hashicorp/terraform/issues/31235))
* AzureRM Backend: This release concludes [the deprecation cycle started in Terraform v1.1](https://www.terraform.io/language/upgrade-guides/1-1#preparation-for-removing-azure-ad-graph-support-in-the-azurerm-backend) for the `azurerm` backend's support of "ADAL" authentication. This backend now supports only "MSAL" (Microsoft Graph) authentication.

    This follows from [Microsoft's own deprecation of Azure AD Graph](https://docs.microsoft.com/en-us/graph/migrate-azure-ad-graph-faq), and so you must follow the migration instructions presented in that Azure documentation to adopt Microsoft Graph and then change your backend configuration to use MSAL authentication before upgrading to Terraform v1.3.
* When making requests to HTTPS servers, Terraform will now reject invalid handshakes that have duplicate extensions, as required by RFC 5246 section 7.4.1.4 and RFC 8446 section 4.2. This may cause new errors when interacting with existing buggy or misconfigured TLS servers, but should not affect correct servers.

    This only applies to requests made directly by Terraform CLI, such as provider installation and remote state storage. Terraform providers are separate programs which decide their own policy for handling of TLS handshakes.
* The following backends, which were deprecated in v1.2.3, have now been removed: `artifactory`, `etcd`, `etcdv3`, `manta`, `swift`. The legacy backend name `azure` has also been removed, because the current Azure backend is named `azurerm`. ([#31711](https://github.com/hashicorp/terraform/issues/31711))

ENHANCEMENTS:

* config: Optional attributes for object type constraints, as described under new features above. ([#31154](https://github.com/hashicorp/terraform/issues/31154))
* config: New built-in function `timecmp` allows determining the ordering relationship between two timestamps while taking potentially-different UTC offsets into account. ([#31687](https://github.com/hashicorp/terraform/pull/31687))
* config: When reporting an error message related to a function call, Terraform will now include contextual information about the signature of the function that was being called, as an aid to understanding why the call might have failed. ([#31299](https://github.com/hashicorp/terraform/issues/31299))
* config: When reporting an error or warning message that isn't caused by values being unknown or marked as sensitive, Terraform will no longer mention any values having those characteristics in the contextual information presented alongside the error. Terraform will still return this information for the small subset of error messages that are specifically about unknown values or sensitive values being invalid in certain contexts. ([#31299](https://github.com/hashicorp/terraform/issues/31299))
* config: `moved` blocks can now describe resources moving to and from modules in separate module packages. ([#31556](https://github.com/hashicorp/terraform/issues/31556))
* `terraform fmt` now accepts multiple target paths, allowing formatting of several individual files at once. ([#28191](https://github.com/hashicorp/terraform/issues/28191))
* `terraform init`: provider installation errors now mention which host Terraform was downloading from ([#31524](https://github.com/hashicorp/terraform/issues/31524))
* CLI: Terraform will report more explicitly when it is proposing to delete an object due to it having moved to a resource instance that is not currently declared in the configuration. ([#31695](https://github.com/hashicorp/terraform/issues/31695))
* CLI: When showing the progress of a remote operation running in Terraform Cloud, Terraform CLI will include information about pre-plan run tasks ([#31617](https://github.com/hashicorp/terraform/issues/31617))
* The AzureRM Backend now only supports MSAL (and Microsoft Graph) and no longer makes use of ADAL (and Azure Active Directory Graph) for authentication ([#31070](https://github.com/hashicorp/terraform/issues/31070))
* The COS backend now supports global acceleration. ([#31425](https://github.com/hashicorp/terraform/issues/31425))
* provider plugin protocol: The Terraform CLI now calls `PlanResourceChange` for compatible providers when destroying resource instances. ([#31179](https://github.com/hashicorp/terraform/issues/31179))
* As an implementation detail of the Terraform Cloud integration, Terraform CLI will now capture and upload [the JSON integration format for state](https://www.terraform.io/internals/json-format#state-representation) along with any newly-recorded state snapshots, which then in turn allows Terraform Cloud to provide that information to API-based external integrations. ([#31698](https://github.com/hashicorp/terraform/issues/31698))

BUG FIXES:

* config: Terraform was not previously evaluating preconditions and postconditions during the apply phase for resource instances that didn't have any changes pending, which was incorrect because the outcome of a condition can potentially be affected by changes to _other_ objects in the configuration. Terraform will now always check the conditions for every resource instance included in a plan during the apply phase, even for resource instances that have "no-op" changes. This means that some failures that would previously have been detected only by a subsequent run will now be detected during the same run that caused them, thereby giving the feedback at the appropriate time. ([#31491](https://github.com/hashicorp/terraform/issues/31491))
* `terraform show -json`: Fixed missing markers for unknown values in the encoding of partially unknown tuples and sets. ([#31236](https://github.com/hashicorp/terraform/issues/31236))
* `terraform output` CLI help documentation is now more consistent with web-based documentation. ([#29354](https://github.com/hashicorp/terraform/issues/29354))
* `terraform init`: Error messages now handle the situation where the underlying HTTP client library does not indicate a hostname for a failed request. ([#31542](https://github.com/hashicorp/terraform/issues/31542))
* `terraform init`: Don't panic if a child module contains a resource with a syntactically-invalid resource type name. ([#31573](https://github.com/hashicorp/terraform/issues/31573))
* CLI: The representation of destroying already-`null` output values in a destroy plan will no longer report them as being deleted, which avoids reporting the deletion of an output value that was already absent. ([#31471](https://github.com/hashicorp/terraform/issues/31471))
* `terraform import`: Better handling of resources or modules that use `for_each`, and situations where data resources are needed to complete the operation. ([#31283](https://github.com/hashicorp/terraform/issues/31283))

EXPERIMENTS:

* This release concludes the `module_variable_optional_attrs` experiment, which started in Terraform v0.14.0. The final design of the optional attributes feature is similar to the experimental form in the previous releases, but with two major differences:
    * The `optional` function-like modifier for declaring an optional attribute now accepts an optional second argument for specifying a default value to use when the attribute isn't set by the caller. If not specified, the default value is a null value of the appropriate type as before.
    * The built-in `defaults` function, previously used to meet the use-case of replacing null values with default values, will not graduate to stable and has been removed. Use the second argument of `optional` inline in your type constraint to declare default values instead.

    If you have any experimental modules that were participating in this experiment, you will need to remove the experiment opt-in and adopt the new syntax for declaring default values in order to migrate your existing module to the stablized version of this feature. If you are writing a shared module for others to use, we recommend declaring that your module requires Terraform v1.3.0 or later to give specific feedback when using the new feature on older Terraform versions, in place of the previous declaration to use the experimental form of this feature:

    ```hcl
    terraform {
      required_version = ">= 1.3.0"
    }
    ```

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
