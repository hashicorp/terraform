## 1.7.0 (Unreleased)

UPGRADE NOTES:

* Input validations are being restored to the state file in this version of Terraform. Due to a state interoperability issue ([#33770](https://github.com/hashicorp/terraform/issues/33770)) in earlier versions, users that require interaction between different minor series should ensure they have upgraded to the following patches:
    * Users of Terraform prior to 1.3.0 are unaffected;
    * Terraform 1.3 series users should upgrade to 1.3.10;
    * Terraform 1.4 series users should upgrade to 1.4.7;
    * Terraform 1.5 series users should upgrade to 1.5.7;
    * Users of Terraform 1.6.0 and later are unaffected.
 
  This is important for users with `terraform_remote_state` data sources reading remote state across different versions of Terraform.

BUG FIXES:

* Ignore potential remote terraform version mismatch when running force-unlock ([#28853](https://github.com/hashicorp/terraform/issues/28853))

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
