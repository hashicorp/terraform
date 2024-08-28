## 1.10.0 (Unreleased)

BUG FIXES:

- The error message for an invalid default value for an input variable now indicates when the problem is with a nested value in a complex data type. ([#35465](https://github.com/hashicorp/terraform/issues/35465))
- Sensitive marks could be incorrectly transferred to nested resource values, causing erroneous changes during a plan ([#35501](https://github.com/hashicorp/terraform/issues/35501))
- Allow unknown `error_message` values to pass the core validate step, so variable validation can be completed later during plan
  ([#35537](https://github.com/hashicorp/terraform/issues/35537))
- Unencoded slashes within GitHub module source refs were being truncated and incorrectly used as subdirectories in the request path ([#35552](https://github.com/hashicorp/terraform/issues/35552))

ENHANCEMENTS:

- The `element` function now accepts negative indices ([#35501](https://github.com/hashicorp/terraform/issues/35501))
- Import block validation has been improved to provide more useful errors and catch more invalid cases during `terraform validate` ([#35543](https://github.com/hashicorp/terraform/issues/35543))
* Performance enhancements for resource evaluation, especially when large numbers of resource instances are involved ([#35558](https://github.com/hashicorp/terraform/issues/35558))

EXPERIMENTS:

Experiments are only enabled in alpha releases of Terraform CLI. The following features are not yet available in stable releases.

- `ephemeral_values`: This [language experiment](https://developer.hashicorp.com/terraform/language/settings#experimental-language-features) introduces a new special kind of value which Terraform allows to change between the plan phase and the apply phase, and between plan/apply rounds. Ephemeral values are never persisted in saved plan files or state snapshots, and so can only be used in parts of the language that don't require values to persist in those artifacts. Ephemeral input values are the main initial example of this concept, allowing the use of input variables to provide dynamic credentials that must change between plan and apply.
- `terraform test` accepts a new option `-junit-xml=FILENAME`. If specified, and if the test configuration is valid enough to begin executing, then Terraform writes a JUnit XML test result report to the given filename, describing similar information as included in the normal test output. ([#34291](https://github.com/hashicorp/terraform/issues/34291))
- The new command `terraform rpcapi` exposes some Terraform Core functionality through an RPC interface compatible with [`go-plugin`](https://github.com/hashicorp/go-plugin). The exact RPC API exposed here is currently subject to change at any time, because it's here primarily as a vehicle to support the [Terraform Stacks](https://www.hashicorp.com/blog/terraform-stacks-explained) private preview and so will be broken if necessary to respond to feedback from private preview participants, or possibly for other reasons. Do not use this mechanism yet outside of Terraform Stacks private preview.
- The experimental "deferred actions" feature, enabled by passing the `-allow-deferral` option to `terraform plan`, permits `count` and `for_each` arguments in `module`, `resource`, and `data` blocks to have unknown values and allows providers to react more flexibly to unknown values. This experiment is under active development, and so it's not yet useful to participate in this experiment

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
