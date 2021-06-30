## 1.1.0 (Unreleased)

NEW FEATURES:

* cli: `terraform add` generates resource configuration templates ([#28874](https://github.com/hashicorp/terraform/issues/28874))
* config: a new `type()` function, only available in `terraform console` ([#28501](https://github.com/hashicorp/terraform/issues/28501))

ENHANCEMENTS:

* config: Terraform now checks the syntax of and normalizes module source addresses (the `source` argument in `module` blocks) during configuration decoding rather than only at module installation time. This is largely just an internal refactoring, but a visible benefit of this change is that the `terraform init` messages about module downloading will now show the canonical module package address Terraform is downloading from, after interpreting the special shorthands for common cases like GitHub URLs. ([#28854](https://github.com/hashicorp/terraform/issues/28854))

BUG FIXES:

* core: Fixed an issue where provider configuration input variables were not properly merging with values in configuration ([#29000](https://github.com/hashicorp/terraform/issues/29000))
* cli: Fixed a crashing bug with some edge-cases when reporting syntax errors that happen to be reported at the position of a newline. ([#29048](https://github.com/hashicorp/terraform/issues/29048))

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
