## 0.14.0 (Unreleased)

ENHANCEMENTS:

* backend/consul: Split state into chunks when outgrowing the limit of the Consul KV store. This allows storing state larger than the Consul 512KB limit. [GH-25856]
* cli: A new global command line option `-chdir=...`, placed before the selected subcommand, instructs Terraform to switch to a different working directory before executing the subcommand. This is similar to switching to a new directory with `cd` before running Terraform, but it avoids changing the state of the calling shell. ([#26087](https://github.com/hashicorp/terraform/issues/26087))
* command: Added an experimental concise diff renderer. By default, Terraform plans now hide most unchanged fields, only displaying the most relevant changes and some identifying context. This experiment can be disabled by setting a `TF_X_CONCISE_DIFF` environment variable to `0`. ([#26187](https://github.com/hashicorp/terraform/issues/26187))
* repl: Improved the renderer used by `terraform console` and for rendering outputs. Terraform now has distinct rendering of lists, sets, and tuples, and correctly renders objects with `null` attribute values. [GH-26189]

BREAKING CHANGES:
* configs: The `version` argument inside provider configuration blocks is deprecated. Instead, use the required_providers setting. ([#26135](https://github.com/hashicorp/terraform/issues/26135))

BUG FIXES:

* backend/consul: Fix bug which prevented state locking when path has trailing `/` [GH-25842]
* build: fix crash with terraform binary on openBSD [#26249]
* command/clistate: return an error on a state unlock failure [[#25729](https://github.com/hashicorp/terraform/issues/25729)] 
* configs: Report an error when provider configuration attributes are incorrectly added to a `required_providers` object. [GH-26184]
* lang/funcs: fix panic when element() is called with a negative offset ([#26079](https://github.com/hashicorp/terraform/issues/26079))
* states/remote: fix `state push -force` to work for all backends ([#26190](https://github.com/hashicorp/terraform/issues/26190))


## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
