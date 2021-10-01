## 1.1.0 (Unreleased)

UPGRADE NOTES:

* Terraform on macOS now requires macOS 10.13 High Sierra or later; Older macOS versions are no longer supported.
* The `terraform graph` command no longer supports `-type=validate` and `-type=eval` options. The validate graph is always the same as the plan graph anyway, and the "eval" graph was just an implementation detail of the `terraform console` command. The default behavior of creating a plan graph should be a reasonable replacement for both of the removed graph modes. (Please note that `terraform graph` is not covered by the Terraform v1.0 compatibility promises, because its behavior inherently exposes Terraform Core implementation details, so we recommend it only for interactive debugging tasks and not for use in automation.)
* `terraform apply` with a previously-saved plan file will now verify that the provider plugin packages used to create the plan fully match the ones used during apply, using the same checksum scheme that Terraform normally uses for the dependency lock file. Previously Terraform was checking consistency of plugins from a plan file using a legacy mechanism which covered only the main plugin executable, not any other files that might be distributed alongside in the plugin package.

    This additional check should not affect typical plugins that conform to the expectation that a plugin package's contents are immutable once released, but may affect a hypothetical in-house plugin that intentionally modifies extra files in its package directory somehow between plan and apply. If you have such a plugin, you'll need to change its approach to store those files in some other location separate from the package directory. This is a minor compatibility break motivated by increasing the assurance that plugins have not been inadvertently or maliciously modified between plan and apply.

NEW FEATURES:

* `terraform plan` and `terraform apply`: When Terraform plans to destroy a resource instance due to it no longer being declared in the configuration, the proposed plan output will now include a note hinting at what situation prompted that proposal, so you can more easily see what configuration change might avoid the object being destroyed. ([#29637](https://github.com/hashicorp/terraform/pull/29637))
* `terraform plan` and `terraform apply`: When Terraform automatically moves a singleton resource instance to index zero or vice-versa in response to adding or removing `count`, it'll report explicitly that it did so as part of the plan output. ([#29605](https://github.com/hashicorp/terraform/pull/29605))
* `terraform add`: The (currently-experimental) `terraform add` generates a starting point for a particular resource configuration. ([#28874](https://github.com/hashicorp/terraform/issues/28874))
* config: a new `type()` function, available only in `terraform console`. ([#28501](https://github.com/hashicorp/terraform/issues/28501))

ENHANCEMENTS:

* config: Terraform now checks the syntax of and normalizes module source addresses (the `source` argument in `module` blocks) during configuration decoding rather than only at module installation time. This is largely just an internal refactoring, but a visible benefit of this change is that the `terraform init` messages about module downloading will now show the canonical module package address Terraform is downloading from, after interpreting the special shorthands for common cases like GitHub URLs. ([#28854](https://github.com/hashicorp/terraform/issues/28854))
* cli: Terraform will now report explicitly in the UI if it automatically moves a resource instance to a new address as a result of adding or removing the `count` argument from an existing resource. For example, if you previously had `resource "aws_subnet" "example"` _without_ `count`, you might have `aws_subnet.example` already bound to a remote object in your state. If you add `count = 1` to that resource then Terraform would previously silently rebind the object to `aws_subnet.example[0]` as part of planning, whereas now Terraform will mention that it did so explicitly in the plan description. ([#29605](https://github.com/hashicorp/terraform/issues/29605))

BUG FIXES:

* core: Fixed an issue where provider configuration input variables were not properly merging with values in configuration ([#29000](https://github.com/hashicorp/terraform/issues/29000))
* cli: Blocks using SchemaConfigModeAttr in the provider SDK can now represented in the plan json output ([#29522](https://github.com/hashicorp/terraform/issues/29522))

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
