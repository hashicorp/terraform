## 0.14.0 (Unreleased)

ENHANCEMENTS:

* cli: A new global command line option `-chdir=...`, placed before the selected subcommand, instructs Terraform to switch to a different working directory before executing the subcommand. This is similar to switching to a new directory with `cd` before running Terraform, but it avoids changing the state of the calling shell. ([#26087](https://github.com/hashicorp/terraform/issues/26087))
* command: Added an experimental concise diff renderer. By default, Terraform plans now hide most unchanged fields, only displaying the most relevant changes and some identifying context. This experiment can be disabled by setting a `TF_X_CONCISE_DIFF` environment variable to `0`. ([#26187](https://github.com/hashicorp/terraform/issues/26187))

BREAKING CHANGES:
* configs: The `version` argument inside provider configuration blocks is deprecated. Instead, use the required_providers setting. ([#26135](https://github.com/hashicorp/terraform/issues/26135))

BUG FIXES:

* command/clistate: return an error on a state unlock failure [[#25729](https://github.com/hashicorp/terraform/issues/25729)] 
* lang/funcs: fix panic when element() is called with a negative offset ([#26079](https://github.com/hashicorp/terraform/issues/26079))
* states/remote: fix `state push -force` to work for all backends ([#26190](https://github.com/hashicorp/terraform/issues/26190))

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
