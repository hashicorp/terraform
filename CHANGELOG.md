## 1.13.5 (Unreleased)

## 1.13.4 (October 15, 2025)


BUG FIXES:

* Fix crash when showing a cloud plan without having a cloud backend ([#37751](https://github.com/hashicorp/terraform/issues/37751))


## 1.13.3 (September 17, 2025)


BUG FIXES:

* variable validation: keep sensitive and ephemeral metadata when evaluating variable conditions. ([#37595](https://github.com/hashicorp/terraform/issues/37595))


## 1.13.2 (September 10, 2025)


BUG FIXES:

* test: Fix the order of execution of cleanup nodes ([#37546](https://github.com/hashicorp/terraform/issues/37546))

* apply: hide sensitive inputs when values have changed between plan and apply ([#37582](https://github.com/hashicorp/terraform/issues/37582))


## 1.13.1 (August 27, 2025)


BUG FIXES:

* Fix regression that caused `terraform test` with zero tests to return a non-zero exit code. ([#37477](https://github.com/hashicorp/terraform/issues/37477))

* terraform test: prevent panic when resolving incomplete references ([#37484](https://github.com/hashicorp/terraform/issues/37484))


## 1.13.0 (August 20, 2025)


NEW FEATURES:

* The new command `terraform stacks` exposes some stack operations through the cli. The available subcommands depend on the stacks plugin implementation. Use `terraform stacks -help` to see available commands. ([#36931](https://github.com/hashicorp/terraform/issues/36931))


ENHANCEMENTS:

* Filesystem functions are now checked for consistent results to catch invalid data during apply ([#37001](https://github.com/hashicorp/terraform/issues/37001))

* Allow successful init when provider constraint matches at least one valid version ([#37137](https://github.com/hashicorp/terraform/issues/37137))

* Performance fix for evaluating high cardinality resources ([#37154](https://github.com/hashicorp/terraform/issues/37154))

*  TF Test: Allow parallel execution of teardown operations ([#37169](https://github.com/hashicorp/terraform/issues/37169))

* `terraform test`: Test authors can now specify definitions for external variables that are referenced within test files directly within the test file itself. ([#37195](https://github.com/hashicorp/terraform/issues/37195))

* `terraform test`: File-level variable blocks can now reference run outputs and other variables." ([#37205](https://github.com/hashicorp/terraform/issues/37205))

* skip redundant comparisons when comparing planned set changes ([#37280](https://github.com/hashicorp/terraform/issues/37280))

* type checking: improve error message on type mismatches. ([#37298](https://github.com/hashicorp/terraform/issues/37298))


BUG FIXES:

* Added a missing warning diagnostic that alerts users when child module contains an ignored `cloud` block. ([#37180](https://github.com/hashicorp/terraform/issues/37180))

* Nested module outputs could lose sensitivity, even when marked as such in the configuration ([#37212](https://github.com/hashicorp/terraform/issues/37212))

* workspace: Updated validation to reject workspaces named "" ([#37267](https://github.com/hashicorp/terraform/issues/37267))

* workspace: Updated the `workspace delete` command to reject `""` as an invalid workspace name ([#37275](https://github.com/hashicorp/terraform/issues/37275))

* plan: truncate invalid or dynamic references in the relevant attributes ([#37290](https://github.com/hashicorp/terraform/issues/37290))

* Test run Parallelism of 1 should not result in deadlock ([#37292](https://github.com/hashicorp/terraform/issues/37292))

* static validation: detect invalid static references via indexes on objects. ([#37298](https://github.com/hashicorp/terraform/issues/37298))

* Fixes resource identity being dropped from state in certain cases ([#37396](https://github.com/hashicorp/terraform/issues/37396))


NOTES:

* The command `terraform rpcapi` is now generally available. It is not intended for public consumption, but exposes certain Terraform operations through an RPC interface compatible with [go-plugin](https://github.com/hashicorp/go-plugin). ([#37067](https://github.com/hashicorp/terraform/issues/37067))


UPGRADE NOTES:

* `terraform test`: External variables referenced within test files should now be accompanied by a `variable` definition block within the test file. This is optional, but users with complex external variables may see error diagnostics without the additional variable definition. ([#37195](https://github.com/hashicorp/terraform/issues/37195))
EXPERIMENTS:

Experiments are only enabled in alpha releases of Terraform CLI. The following features are not yet available in stable releases.

- The experimental "deferred actions" feature, enabled by passing the `-allow-deferral` option to `terraform plan`, permits `count` and `for_each` arguments in `module`, `resource`, and `data` blocks to have unknown values and allows providers to react more flexibly to unknown values.

## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

- [v1.12](https://github.com/hashicorp/terraform/blob/v1.12/CHANGELOG.md)
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
