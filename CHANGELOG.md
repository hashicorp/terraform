## 1.14.0 (Unreleased)


NEW FEATURES:

* **List Resources**: List resources can be defined in `*.tfquery.hcl` files and allow querying and filterting existing infrastructure.

* A new Terraform command `terraform query`: Executes list operations against existing infrastructure and displays the results. The command can optionally generate configuration for importing results into Terraform.

* A new GenerateResourceConfiguration RPC allows providers to create more precise configuration values during import. ([#37515](https://github.com/hashicorp/terraform/issues/37515))

* New top-level Actions block: Actions are provider defined and meant to codify use cases outside the normal CRUD model in your Terraform configuration. Providers can define Actions like `aws_lambda_invoke` or `aws_cloudfront_create_invalidation` that do something imparative outside of Terraforms normal CRUD model. You can configure such a side-effect with an action block and have actions triggered through the lifecycle of a resource or through passing the `-invoke` CLI flag. ([#37553](https://github.com/hashicorp/terraform/issues/37553))


ENHANCEMENTS:

* terraform test: expected diagnostics will be included in test output when running in verbose mode" ([#37362](https://github.com/hashicorp/terraform/issues/37362))

* terraform test: ignore prevent_destroy attribute during when cleaning up tests" ([#37364](https://github.com/hashicorp/terraform/issues/37364))

* `terraform stacks` command support for `-help` flag ([#37645](https://github.com/hashicorp/terraform/issues/37645))

* query: support offline validation of query files via -query flag in the validate command ([#37671](https://github.com/hashicorp/terraform/issues/37671))

* Updates to support the AWS European Sovereign Cloud ([#37721](https://github.com/hashicorp/terraform/issues/37721))


BUG FIXES:

* Retrieve all workspace variables while doing a `terraform import`, include variables inherited from variable sets but not overwritten by the workspace. ([#37241](https://github.com/hashicorp/terraform/issues/37241))

* Fix OSS backend proxy support by adding a proxy layer for OSS backend operations. Resolves hashicorp/terraform#36897. ([#36897](https://github.com/hashicorp/terraform/issues/36897))

* console and test: return explicit diagnostics when referencing resources that were not included in the most recent operation. ([#37663](https://github.com/hashicorp/terraform/issues/37663))

* query: generate unique resource identifiers for results of expanded list resources ([#37681](https://github.com/hashicorp/terraform/issues/37681))

* The CLI now summarizes the number of actions invoked during `terraform apply`, matching the plan output. ([#37689](https://github.com/hashicorp/terraform/issues/37689))


UPGRADE NOTES:

* The parallelism of Terraform operations within container runtimes may be reduced depending on the CPU bandwidth limit setting. ([#37436](https://github.com/hashicorp/terraform/issues/37436))

* Building Terraform 1.14 requires macOS Monterey or later (due to being built on Go 1.25 which imposes these requirements) ([#37436](https://github.com/hashicorp/terraform/issues/37436))


EXPERIMENTS:

Experiments are only enabled in alpha releases of Terraform CLI. The following features are not yet available in stable releases.

- The experimental "deferred actions" feature, enabled by passing the `-allow-deferral` option to `terraform plan`, permits `count` and `for_each` arguments in `module`, `resource`, and `data` blocks to have unknown values and allows providers to react more flexibly to unknown values.
- `terraform test cleanup`: The experimental `test cleanup` command. In experimental builds of Terraform, a manifest file and state files for each failed cleanup operation during test operations are saved within the `.terraform` local directory. The `test cleanup` command will attempt to clean up the local state files left behind automatically, without requiring manual intervention.
- `terraform test`: `backend` blocks and `skip_cleanup` attributes:
  - Test authors can now specify `backend` blocks within `run` blocks in Terraform Test files. Run blocks with `backend` blocks will load state from the specified backend instead of starting from empty state on every execution. This allows test authors to keep long-running test infrastructure alive between test operations, saving time during regular test operations.
  - Test authors can now specify `skip_cleanup` attributes within test files and within run blocks. The `skip_cleanup` attribute tells `terraform test` not to clean up state files produced by run blocks with this attribute set to true. The state files for affected run blocks will be written to disk within the `.terraform` directory, where they can then be cleaned up manually using the also experimental `terraform test cleanup` command.

## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

- [v1.13](https://github.com/hashicorp/terraform/blob/v1.13/CHANGELOG.md)
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
