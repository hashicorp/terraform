## 1.7.3 (Unreleased)

BUG FIXES: 

* `terraform test`: Fix crash when dynamic-typed attributes are not assigned values in mocks. ([#34610](https://github.com/hashicorp/terraform/pull/34511))
* don't panic when file provisioner source is null. ([#34621](https://github.com/hashicorp/terraform/pull/34621))
* throw helpful error message if import block is configured with id "" ([34625](https://github.com/hashicorp/terraform/pull/34625))

## 1.7.2 (January 31, 2024)

BUG FIXES:

* backend/s3: No longer returns error when IAM user or role does not have access to the default workspace prefix `env:`. ([#34511](https://github.com/hashicorp/terraform/pull/34511))
* cloud: When triggering a run, the .terraform/modules directory was being excluded from the configuration upload causing Terraform Cloud to try (and sometimes fail) to re-download the modules. ([#34543](https://github.com/hashicorp/terraform/issues/34543))

ENHANCEMENTS:

* `terraform fmt`: Terraform mock data files (`.tfmock.hcl`) will now be included when executing the format command. ([#34580](https://github.com/hashicorp/terraform/issues/34580))
* Add additional diagnostics when a generated provider block that fails schema validation requires explicit configuration. ([#34595](https://github.com/hashicorp/terraform/issues/34595))

## 1.7.1 (January 24, 2024)

BUG FIXES:

* `terraform test`: Fix crash when referencing variables or functions within the file level `variables` block. ([#34531](https://github.com/hashicorp/terraform/issues/34531))
* `terraform test`: Fix crash when `override_module` block was missing the `outputs` attribute. ([#34563](https://github.com/hashicorp/terraform/issues/34563))

## 1.7.0 (January 17, 2024)

UPGRADE NOTES:

* Input validations are being restored to the state file in this version of Terraform. Due to a state interoperability issue ([#33770](https://github.com/hashicorp/terraform/issues/33770)) in earlier versions, users that require interaction between different minor series should ensure they have upgraded to the following patches:
    * Users of Terraform prior to 1.3.0 are unaffected;
    * Terraform 1.3 series users should upgrade to 1.3.10;
    * Terraform 1.4 series users should upgrade to 1.4.7;
    * Terraform 1.5 series users should upgrade to 1.5.7;
    * Users of Terraform 1.6.0 and later are unaffected.
 
  This is important for users with `terraform_remote_state` data sources reading remote state across different versions of Terraform.
* `nonsensitive` function no longer raises an error when applied to a value that is already non-sensitive. ([#33856](https://github.com/hashicorp/terraform/issues/33856))
* `terraform graph` now produces a simplified graph describing only relationships between resources by default, for consistency with the granularity of information returned by other commands that emphasize resources as the main interesting object type and de-emphasize the other "glue" objects that connect them.

    The type of graph that earlier versions of Terraform produced by default is still available with explicit use of the `-type=plan` option, producing an approximation of the real dependency graph Terraform Core would use to construct a plan.
* `terraform test`: Simplify the ordering of destroy operations during test cleanup to simple reverse run block order. ([#34293](https://github.com/hashicorp/terraform/issues/34293))

* backend/s3: The `use_legacy_workflow` argument now defaults to `false`. The backend will now search for credentials in the same order as the default provider chain in the AWS SDKs and AWS CLI. To revert to the legacy credential provider chain ordering, set this value to `true`. This argument, and the ability to use the legacy workflow, is deprecated. To encourage consistency with the AWS SDKs, this argument will be removed in a future minor version.

NEW FEATURES:

* `terraform test`: Providers, modules, resources, and data sources can now be mocked during executions of `terraform test`. The following new blocks have been introduced within `.tftest.hcl` files:

    * `mock_provider`: Can replace provider instances with mocked providers, allowing tests to execute in `command = apply` mode without requiring a configured cloud provider account and credentials. Terraform will create fake resources for mocked providers and maintain them in state for the lifecycle of the given test file.
    * `override_resource`: Specific resources can be overridden so Terraform will create a fake resource with custom values instead of creating infrastructure for the overridden resource.
    * `override_data`: Specific data sources can be overridden so data can be imported into tests without requiring real infrastructure to be created externally first.
    * `override_module`: Specific modules can be overridden in their entirety to give greater control over the returned outputs without requiring in-depth knowledge of the module itself.
 
* `removed` block for refactoring modules: Module authors can now record in source code when a resource or module call has been removed from configuration, and can inform Terraform whether the corresponding object should be deleted or simply removed from state.
  
  This effectively provides a configuration-driven workflow to replace `terraform state rm`. Removing an object from state is a new type of action which is planned and applied like any other. The `terraform state rm` command will remain available for scenarios in which directly modifying the state file is appropriate.

BUG FIXES:

* Ignore potential remote terraform version mismatch when running force-unlock ([#28853](https://github.com/hashicorp/terraform/issues/28853))
* Exit Dockerfile build script early on `cd` failure. ([#34128](https://github.com/hashicorp/terraform/issues/34128))
* `terraform test`: Stop attempting to destroy run blocks that have no actual infrastructure to destroy. This fixes an issue where attempts to destroy "verification" run blocks that load only data sources would fail if the underlying infrastructure referenced by the run blocks had already been destroyed. ([#34331](https://github.com/hashicorp/terraform/pull/34331))
* `terraform test`: Improve error message for invalid run block names. ([#34469](https://github.com/hashicorp/terraform/pull/34469))
* `terraform test`: Fix bug where outputs in "empty" modules were not available to the assertions from Terraform test files. ([#34482](https://github.com/hashicorp/terraform/pull/34482))
* security: Upstream patch to mitigate the security advisory CVE-2023-48795, which potentially affects `local-exec` and `file` provisioners connecting to remote hosts using SSH. ([#34426](https://github.com/hashicorp/terraform/issues/34426))

ENHANCEMENTS:

* `terraform test`: Providers defined within test files can now reference variables from their configuration that are defined within the test file. ([#34069](https://github.com/hashicorp/terraform/issues/34069))
* `terraform test`: Providers defined within test files can now reference outputs from run blocks. ([#34118](https://github.com/hashicorp/terraform/issues/34118))
* `terraform test`: Terraform functions are now available within variables and provider blocks within test files. ([#34204](https://github.com/hashicorp/terraform/issues/34204))
* `terraform test`: Terraform will now load variables from any `terraform.tfvars` within the testing directory, and apply the variable values to tests within the same directory. ([#34341](https://github.com/hashicorp/terraform/pull/34341))
* `terraform graph`: Now produces a simplified resources-only graph by default. ([#34288](https://github.com/hashicorp/terraform/pull/34288))
* `terraform console`: Now supports a `-plan` option which allows evaluating expressions against the planned new state, rather than against the prior state. This provides a more complete set of values for use in console expressions, at the expense of a slower startup time due first calculating the plan. ([#34342](https://github.com/hashicorp/terraform/issues/34342))
* `import`: `for_each` can now be used to expand the `import` block to handle multiple resource instances ([#33932](https://github.com/hashicorp/terraform/issues/33932))
* If the proposed change for a resource instance is rejected either due to a `postcondition` block or a `prevent_destroy` setting, Terraform will now include that proposed change in the plan output alongside the relevant error, whereas before the error would _replace_ the proposed change in the output. ([#34312](https://github.com/hashicorp/terraform/issues/34312))
* `.terraformignore`: improve performance when ignoring large directories ([#34400](https://github.com/hashicorp/terraform/pull/34400))

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.6](https://github.com/hashicorp/terraform/blob/v1.6/CHANGELOG.md)
* [v1.5](https://github.com/hashicorp/terraform/blob/v1.5/CHANGELOG.md)
* [v1.4](https://github.com/hashicorp/terraform/blob/v1.4/CHANGELOG.md)
* [v1.3](https://github.com/hashicorp/terraform/blob/v1.3/CHANGELOG.md)
* [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
