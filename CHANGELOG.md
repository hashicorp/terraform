## 0.14.0 (Unreleased)

UPGRADE NOTES:
* configs: The `version` argument inside provider configuration blocks has been documented as deprecated since Terraform 0.12. As of 0.14 it will now also generate an explicit deprecation warning. To avoid the warning, use [provider requirements](https://www.terraform.io/docs/configuration/provider-requirements.html) declarations instead. ([#26135](https://github.com/hashicorp/terraform/issues/26135))

ENHANCEMENTS:

* `terraform plan` and `terraform apply`: Added an experimental concise diff renderer. By default, Terraform plans now hide most unchanged fields, only displaying the most relevant changes and some identifying context. This experiment can be disabled by setting a `TF_X_CONCISE_DIFF` environment variable to `0`. ([#26187](https://github.com/hashicorp/terraform/issues/26187))
* cli: A new global command line option `-chdir=...`, placed before the selected subcommand, instructs Terraform to switch to a different working directory before executing the subcommand. This is similar to switching to a new directory with `cd` before running Terraform, but it avoids changing the state of the calling shell. ([#26087](https://github.com/hashicorp/terraform/issues/26087))
* config: Added `alltrue` function, which returns `true` if all elements in the given collection are `true`. This is primarily intended to make it easier to write variable validation conditions which operate on collections. ([#25656](https://github.com/hashicorp/terraform/issues/25656))
* core: `terraform plan` no longer uses a separate refresh phase, all resources are updated on-demand during planning ([#26270](https://github.com/hashicorp/terraform/issues/26270))
* `terraform console`: Now has distinct rendering of lists, sets, and tuples, and correctly renders objects with `null` attribute values. ([#26189](https://github.com/hashicorp/terraform/issues/26189))
* `terraform login`: Added support for OAuth2 application scopes. ([#26239](https://github.com/hashicorp/terraform/issues/26239))
* backend/consul: Split state into chunks when outgrowing the limit of the Consul KV store. This allows storing state larger than the Consul 512KB limit. ([#25856](https://github.com/hashicorp/terraform/issues/25856))

BUG FIXES:

* backend/consul: Fix bug which prevented state locking when path has trailing `/` ([#25842](https://github.com/hashicorp/terraform/issues/25842))
* build: Fix crash with terraform binary on OpenBSD. ([#26249](https://github.com/hashicorp/terraform/issues/26249)
* command/clistate: return an error on a state unlock failure [[#25729](https://github.com/hashicorp/terraform/issues/25729)] 
* command/taint: If the configuration's `required_version` constraint is not met, the `taint` subcommand will now correctly exit early. [GH-26345]
* configs: Report an error when provider configuration attributes are incorrectly added to a `required_providers` object. ([#26184](https://github.com/hashicorp/terraform/issues/26184))
* core: Errors with data sources reading old data during refresh, failing to refresh, and not appearing to wait on resource dependencies are fixed by updates to the data source lifecycle and the merging of refresh and plan ([#26270](https://github.com/hashicorp/terraform/issues/26270))
* lang/funcs: fix panic when element() is called with a negative offset ([#26079](https://github.com/hashicorp/terraform/issues/26079))
* states/remote: fix `state push -force` to work for all backends ([#26190](https://github.com/hashicorp/terraform/issues/26190))


## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
