## 1.7.0 (Unreleased)

UPGRADE NOTES:

* Input validations are being restored to the state file in this version of Terraform. Due to a state interoperability issue ([#33770](https://github.com/hashicorp/terraform/issues/33770)) in earlier versions, users that require interaction between different minor series should ensure they have upgraded to the following patches:
    * Users of Terraform prior to 1.3.0 are unaffected;
    * Terraform 1.3 series users should upgrade to 1.3.10;
    * Terraform 1.4 series users should upgrade to 1.4.7;
    * Terraform 1.5 series users should upgrade to 1.5.7;
    * Users of Terraform 1.6.0 and later are unaffected.
 
  This is important for users with `terraform_remote_state` data sources reading remote state across different versions of Terraform.
* `nonsensitive` function no longer errors when applied to values that are already not sensitive. ([#33856](https://github.com/hashicorp/terraform/issues/33856))

BUG FIXES:

* Ignore potential remote terraform version mismatch when running force-unlock ([#28853](https://github.com/hashicorp/terraform/issues/28853))
* Exit Dockerfile build script early on `cd` failure. ([#34128](https://github.com/hashicorp/terraform/issues/34128))

ENHANCEMENTS:

* `terraform test`: Providers defined within test files can now reference variables from their configuration that are defined within the test file. ([#34069](https://github.com/hashicorp/terraform/issues/34069))
* `terraform test`: Providers defined within test files can now reference outputs from run blocks. ([#34118](https://github.com/hashicorp/terraform/issues/34118))
* `import`: `for_each` can now be used to expand the `import` block to handle multiple resource instances ([#33932](https://github.com/hashicorp/terraform/issues/33932))

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
