## 1.10.4 (January 8, 2025)


BUG FIXES:

* type conversion: Empty map conversions now return correct type information ([#36262](https://github.com/hashicorp/terraform/issues/36262))

* `terraform console`: Fix crash when printing ephemeral values ([#36267](https://github.com/hashicorp/terraform/issues/36267))


## 1.10.3 (December 18, 2024)

BUG FIXES:
- Terraform could panic when encountering an error during plan encoding ([#36212](https://github.com/hashicorp/terraform/issues/36212))

## 1.10.2 (December 11, 2024)

BUG FIXES:
- cli: variables in an auto-loaded tfvars file which were overridden during plan incorrectly show as changed during apply ([#36180](https://github.com/hashicorp/terraform/issues/36180))

## 1.10.1 (December 4, 2024)

BUG FIXES:
- cli: Complex variables values set via environment variables were parsed incorrectly during apply ([#36121](https://github.com/hashicorp/terraform/issues/36121))
- config: `templatefile` would panic if given and entirely unknown map of variables ([#36118](https://github.com/hashicorp/terraform/issues/36118))
- config: `templatefile` would panic if the variables map contains marked values ([#36127](https://github.com/hashicorp/terraform/issues/36127))
- config: Remove constraint that an expanded resource block must only be used in conjunction with imports using `for_each` ([#36119](https://github.com/hashicorp/terraform/issues/36119))
- backend/s3: Lock files could not be written to buckets with object locking enabled ([#36120](https://github.com/hashicorp/terraform/issues/36120))

## 1.10.0 (November 27, 2024)

NEW FEATURES:
 - **Ephemeral resources**: Ephemeral resources are read anew during each phase of Terraform evaluation, and cannot be persisted to state storage. Ephemeral resources always produce ephemeral values.
 - **Ephemeral values**: Input variables and outputs can now be defined as ephemeral. Ephemeral values may only be used in certain contexts in Terraform configuration, and are not persisted to the plan or state files.
    - **`ephemeralasnull` function**: a function takes a value of any type and returns a similar value of the same type with any ephemeral values replaced with non-ephemeral null values and all non-ephemeral values preserved.

BUG FIXES:

- The `secret_suffix` in the `kubernetes` backend now includes validation to prevent errors when the `secret_suffix` ends with a number ([#35666](https://github.com/hashicorp/terraform/pull/35666)).
- The error message for an invalid default value for an input variable now indicates when the problem is with a nested value in a complex data type. ([#35465](https://github.com/hashicorp/terraform/issues/35465))
- Sensitive marks could be incorrectly transferred to nested resource values, causing erroneous changes during a plan ([#35501](https://github.com/hashicorp/terraform/issues/35501))
- Allow unknown `error_message` values to pass the core validate step, so variable validation can be completed later during plan
  ([#35537](https://github.com/hashicorp/terraform/issues/35537))
- Unencoded slashes within GitHub module source refs were being truncated and incorrectly used as subdirectories in the request path ([#35552](https://github.com/hashicorp/terraform/issues/35552))
- Terraform refresh-only plans with output only changes are now applyable. ([#35812](https://github.com/hashicorp/terraform/issues/35812))
- Postconditions referencing `self` with many instances could encounter an error during evaluation ([#35895](https://github.com/hashicorp/terraform/issues/35895))
- The `plantimestamp()` function would return an invalid date during validation ([#35902](https://github.com/hashicorp/terraform/issues/35902))
- Updates to resources which were forced to use create_before_destroy could lose that flag in the state temporarily and cause cycles if immediately removed from the configuration ([#35966](https://github.com/hashicorp/terraform/issues/35966))
- backend/cloud: Prefer KV tags, even when tags are defined as set ([#35937](https://github.com/hashicorp/terraform/pull/35937))
- Simplify config generation (`plan -generate-config-out`) for string attributes that contain primitive types (e.g. numbers or booleans) ([#35984](https://github.com/hashicorp/terraform/pull/35984))
- config: `issensitive` could incorrectly assert that an unknown value was not sensitive during plan, but later became sensitive during apply, causing failures where changes did not match the planned result ([#36012](https://github.com/hashicorp/terraform/issues/36012))
- config: The evaluation of conditional expressions and for expression in HCL could lose marks with certain combinations of unknown values ([#36017](https://github.com/hashicorp/terraform/issues/36017))


ENHANCEMENTS:

- The `element` function now accepts negative indices ([#35501](https://github.com/hashicorp/terraform/issues/35501))
- Import block validation has been improved to provide more useful errors and catch more invalid cases during `terraform validate` ([#35543](https://github.com/hashicorp/terraform/issues/35543))
- Performance enhancements for resource evaluation, especially when large numbers of resource instances are involved ([#35558](https://github.com/hashicorp/terraform/issues/35558))
- The `plan`, `apply`, and `refresh` commands now produce a deprecated warning when using the `-state` flag. Instead use the `path` attribute within the `local` backend to modify the state file. ([#35660](https://github.com/hashicorp/terraform/issues/35660))
- backend/cos: Add new auth for Tencent Cloud backend ([#35888](https://github.com/hashicorp/terraform/pull/35888))


UPGRADE NOTES:

- backend/s3: Removes deprecated attributes for assuming IAM role. Must use the `assume_role` block ([#35721](https://github.com/hashicorp/terraform/issues/35721))
- backend/s3: The s3 backend now supports S3 native state locking. When used with DynamoDB-based locking, locks will be acquired from both sources. In a future minor release of Terraform the DynamoDB locking mechanism and associated arguments will be deprecated. ([#35661](https://github.com/hashicorp/terraform/issues/35661))
- `moved`: Moved blocks now respect reserved keywords when parsing resource addresses. Configurations that reference resources with type names that match top level blocks and keywords from `moved` blocks will need to prepend the `resource.` identifier to these references. ([#35850](https://github.com/hashicorp/terraform/issues/35850))
- config: In order to ensure consistency in results from HCL conditional expressions, marks must be combined from all values within the expression to avoid losing mark information. This typically improves accuracy when validating configuration, but users may see sensitive results where they were lost previously.

## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

- [v1.9](https://github.com/hashicorp/terraform/blob/v1.9/CHANGELOG.md)
- [v1.8](https://github.com/hashicorp/terraform/blob/v1.8/CHANGELOG.md)
- [v1.7](https://github.com/hashicorp/terraform/blob/v1.7/CHANGELOG.md)
- [v1.6](https://github.com/hashicorp/terraform/blob/v1.6/CHANGELOG.md)
- [v1.5](https://github.com/hashicorp/terraform/blob/v1.5/CHANGELOG.md)
- [v1.4](https://github.com/hashicorp/terraform/blob/v1.4/CHANGELOG.md)
- [v1.3](https://github.com/hashicorp/terraform/blob/v1.3/CHANGELOG.md)
- [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
- [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
- [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
- [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
- [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
- [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
- [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
- [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
