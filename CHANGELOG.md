## 1.14.6 (February 25, 2026)


BUG FIXES:

* terraform test: return error when provider config is invalid ([#38084](https://github.com/hashicorp/terraform/issues/38084))


## 1.14.5 (February 11, 2026)


BUG FIXES:

* Fixed an issue where terraform stacks validate was failing to resolve relative paths for modules ([#38025](https://github.com/hashicorp/terraform/issues/38025))


## 1.14.4 (January 28, 2026)


BUG FIXES:

* backend: Fix nil pointer dereference crash during `terraform init` when the destination backend returns an error ([#38027](https://github.com/hashicorp/terraform/issues/38027))

* Fixes an issue where any warning diagnostics generated during terraform query execution failed to render in the cloud backend session ([#38040](https://github.com/hashicorp/terraform/issues/38040))

* actions in modules without instances failed the plan graph ([#38089](https://github.com/hashicorp/terraform/issues/38089))


## 1.14.3 (December 17, 2025)


BUG FIXES:

* stacks: change absolute paths in path.module/path.root to be relative, as documented ([#37982](https://github.com/hashicorp/terraform/issues/37982))


## 1.14.2 (December 11, 2025)


ENHANCEMENTS:

* Add component registry source resolution support to Terraform Stacks ([#37888](https://github.com/hashicorp/terraform/issues/37888))


BUG FIXES:

* stacks: surface runtime issues with local values to user during plan ([#37980](https://github.com/hashicorp/terraform/issues/37980))

* resource instance apply failures should not cause the resource instance state to be empty. ([#37981](https://github.com/hashicorp/terraform/issues/37981))
## 1.14.1 (December 3, 2025)


BUG FIXES:

* test: allow ephemeral outputs in root modules ([#37813](https://github.com/hashicorp/terraform/issues/37813))

* Combinations of replace_triggered_by and -replace could result in some instances not being replaced ([#37833](https://github.com/hashicorp/terraform/issues/37833))

* providers lock: include providers required by terraform test ([#37851](https://github.com/hashicorp/terraform/issues/37851))

* Set state information in the proto request for the `GenerateResourceConfig` RPC ([#37896](https://github.com/hashicorp/terraform/issues/37896))

* actions: make after_create & after_update actions run after the resource has applied ([#37936](https://github.com/hashicorp/terraform/issues/37936))


## 1.14.0 (November 19, 2025)


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

* Allow filesystem functions to return inconsistent results when evaluated within provider configuration ([#37854](https://github.com/hashicorp/terraform/issues/37854))

* query: improve error handling for missing identity schemas ([#37863](https://github.com/hashicorp/terraform/issues/37863))


UPGRADE NOTES:

* The parallelism of Terraform operations within container runtimes may be reduced depending on the CPU bandwidth limit setting. ([#37436](https://github.com/hashicorp/terraform/issues/37436))

* Building Terraform 1.14 requires macOS Monterey or later (due to being built on Go 1.25 which imposes these requirements) ([#37436](https://github.com/hashicorp/terraform/issues/37436))


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
