## 1.12.0 (Unreleased)


ENHANCEMENTS:

* Terraform Test command now accepts a -parallelism=n option, which sets the number of parallel operations in a test run's plan/apply operation. ([#34237](https://github.com/hashicorp/terraform/issues/34237))

* Logical binary operators can now short-circuit ([#36224](https://github.com/hashicorp/terraform/issues/36224))

* Terraform Test: Runs can now be annotated for possible parallel execution. ([#34180](https://github.com/hashicorp/terraform/issues/34180))

* Allow terraform init when tests are present but no configuration files are directly inside the current directory ([#35040](https://github.com/hashicorp/terraform/issues/35040))

* Terraform Test: Continue subsequent test execution when an expected failure is not encountered. ([#34969](https://github.com/hashicorp/terraform/issues/34969))

* Produce detailed diagnostic objects when test run assertions fail ([#34428](https://github.com/hashicorp/terraform/issues/34428))

* Improved elapsed time display in UI Hook to show minutes and seconds in `mm:ss` format. ([#36368](https://github.com/hashicorp/terraform/issues/36368))


BUG FIXES:

* Refreshed state was not used in the plan for orphaned resource instances ([#36394](https://github.com/hashicorp/terraform/issues/36394))

* Fixes malformed Terraform version error when the remote backend reads a remote workspace that specifies a Terraform version constraint. ([#36356](https://github.com/hashicorp/terraform/issues/36356))

* Changes to the order of sensitive attributes in the state format would erroneously indicate a plan contained changes when there were none. ([#36465](https://github.com/hashicorp/terraform/issues/36465))

* Avoid reporting duplicate attribute-associated diagnostics, such as "Available Write-only Attribute Alternative" ([#36579](https://github.com/hashicorp/terraform/issues/36579))

* Fixes unintended exit of CLI when using the remote backend and applying with post-plan tasks configured in HCP Terraform ([#36655](https://github.com/hashicorp/terraform/issues/36655))


UPGRADE NOTES:

* On Linux, Terraform now requires Linux kernel version 3.2 or later; support for previous versions has been discontinued. ([#36478](https://github.com/hashicorp/terraform/issues/36478))


EXPERIMENTS:

Experiments are only enabled in alpha releases of Terraform CLI. The following features are not yet available in stable releases.

- The new command `terraform rpcapi` exposes some Terraform Core functionality through an RPC interface compatible with [`go-plugin`](https://github.com/hashicorp/go-plugin). The exact RPC API exposed here is currently subject to change at any time, because it's here primarily as a vehicle to support the [Terraform Stacks](https://www.hashicorp.com/blog/terraform-stacks-explained) private preview and so will be broken if necessary to respond to feedback from private preview participants, or possibly for other reasons. Do not use this mechanism yet outside of Terraform Stacks private preview.
- The experimental "deferred actions" feature, enabled by passing the `-allow-deferral` option to `terraform plan`, permits `count` and `for_each` arguments in `module`, `resource`, and `data` blocks to have unknown values and allows providers to react more flexibly to unknown values. This experiment is under active development, and so it's not yet useful to participate in this experiment

## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

- [v1.11](https://github.com/hashicorp/terraform/blob/v1.11/CHANGELOG.md)
- [v1.10](https://github.com/hashicorp/terraform/blob/v1.10/CHANGELOG.md)
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
