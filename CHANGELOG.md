## 1.3.0 (Unreleased)

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

    Consumers of the JSON output format expecting on the `after_unknown` field to be only `false` or `true` should be updated to support [the change representation](https://www.terraform.io/internals/json-format#change-representation) described in the documentation, and as was already used for resource changes. ([#31235](https://github.com/hashicorp/terraform/issues/31235))

ENHANCEMENTS:

* config: Optional attributes for object type constraints, as described under new features above. ([#31154](https://github.com/hashicorp/terraform/issues/31154))
* `terraform fmt` now accepts multiple target paths, allowing formatting of several individual files at once. ([#28191](https://github.com/hashicorp/terraform/issues/28191))
* When reporting an error message related to a function call, Terraform will now include contextual information about the signature of the function that was being called, as an aid to understanding why the call might have failed. ([#31299](https://github.com/hashicorp/terraform/issues/31299))
* When reporting an error or warning message that isn't caused by values being unknown or marked as sensitive, Terraform will no longer mention any values having those characteristics in the contextual information presented alongside the error. Terraform will still return this information for the small subset of error messages that are specifically about unknown values or sensitive values being invalid in certain contexts. ([#31299](https://github.com/hashicorp/terraform/issues/31299))
* The Terraform CLI now calls `PlanResourceChange` for compatible providers when destroying resource instances. ([#31179](https://github.com/hashicorp/terraform/issues/31179))
* The AzureRM Backend now only supports MSAL (and Microsoft Graph) and no longer makes use of ADAL (and Azure Active Directory Graph) for authentication ([#31070](https://github.com/hashicorp/terraform/issues/31070))
* The COS backend now supports global acceleration. ([#31425](https://github.com/hashicorp/terraform/issues/31425))
* providercache: include host in provider installation error ([#31524](https://github.com/hashicorp/terraform/issues/31524))
* refactoring: `moved` blocks can now be used to move resources to and from external modules ([#31556](https://github.com/hashicorp/terraform/issues/31556))

BUG FIXES:

* config: Terraform was not previously evaluating preconditions and postconditions during the apply phase for resource instances that didn't have any changes pending, which was incorrect because the outcome of a condition can potentially be affected by changes to _other_ objects in the configuration. Terraform will now always check the conditions for every resource instance included in a plan during the apply phase, even for resource instances that have "no-op" changes. This means that some failures that would previously have been detected only by a subsequent run will now be detected during the same run that caused them, thereby giving the feedback at the appropriate time. ([#31491](https://github.com/hashicorp/terraform/issues/31491))
* `terraform show -json`: Fixed missing unknown markers in the encoding of partially unknown tuples and sets. ([#31236](https://github.com/hashicorp/terraform/issues/31236))
* `terraform output` CLI help documentation is now more consistent with web-based documentation. ([#29354](https://github.com/hashicorp/terraform/issues/29354))
* getproviders: account for occasionally missing Host header in errors ([#31542](https://github.com/hashicorp/terraform/issues/31542))
* core: Do not create "delete" changes for nonexistent outputs ([#31471](https://github.com/hashicorp/terraform/issues/31471))
* configload: validate implied provider names in submodules to avoid crash ([#31573](https://github.com/hashicorp/terraform/issues/31573))
* core: `import` fails when resources or modules are expanded with for each, or input from data sources is required ([#31283](https://github.com/hashicorp/terraform/issues/31283))

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
