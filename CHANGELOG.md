## 0.14.0 (Unreleased)

ENHANCEMENTS:

* cli: A new global command line option `-chdir=...`, placed before the selected subcommand, instructs Terraform to switch to a different working directory before executing the subcommand. This is similar to switching to a new directory with `cd` before running Terraform, but it avoids changing the state of the calling shell. [GH-26087]

BREAKING CHANGES:
* configs: The `version` argument inside provider configuration blocks is deprecated. Instead, use the required_providers setting. [GH-26135]

BUG FIXES:

* command/clistate: return an error on a state unlock failure [GH-25729] 
* lang/funcs: fix panic when element() is called with a negative offset [GH-26079]

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
