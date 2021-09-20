## 1.1.0 (Unreleased)

UPGRADE NOTES:

* Terraform on macOS now requires macOS 10.13 High Sierra or later; Older macOS versions are no longer supported.
* The `terraform graph` command no longer supports `-type=validate` and `-type=eval` options. The validate graph is always the same as the plan graph anyway, and the "eval" graph was just an implementation detail of the `terraform console` command. The default behavior of creating a plan graph should be a reasonable replacement for both of the removed graph modes. (Please note that `terraform graph` is not covered by the Terraform v1.0 compatibility promises, because its behavior inherently exposes Terraform Core implementation details, so we recommend it only for interactive debugging tasks and not for use in automation.)

NEW FEATURES:

* cli: The (currently-experimental) `terraform add` generates a starting point for a particular resource configuration. ([#28874](https://github.com/hashicorp/terraform/issues/28874))
* config: a new `type()` function, only available in `terraform console` ([#28501](https://github.com/hashicorp/terraform/issues/28501))

ENHANCEMENTS:

* config: Terraform now checks the syntax of and normalizes module source addresses (the `source` argument in `module` blocks) during configuration decoding rather than only at module installation time. This is largely just an internal refactoring, but a visible benefit of this change is that the `terraform init` messages about module downloading will now show the canonical module package address Terraform is downloading from, after interpreting the special shorthands for common cases like GitHub URLs. ([#28854](https://github.com/hashicorp/terraform/issues/28854))
* cli: Terraform will now report explicitly in the UI if it automatically moves a resource instance to a new address as a result of adding or removing the `count` argument from an existing resource. For example, if you previously had `resource "aws_subnet" "example"` _without_ `count`, you might have `aws_subnet.example` already bound to a remote object in your state. If you add `count = 1` to that resource then Terraform would previously silently rebind the object to `aws_subnet.example[0]` as part of planning, whereas now Terraform will mention that it did so explicitly in the plan description. [GH-29605]

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
