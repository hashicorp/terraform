## 0.15.0 (Unreleased)

ENHANCEMENTS:

* cli: Improved support for Windows console UI on Windows 10, including bold colors and underline for HCL diagnostics. [GH-26588]
* cli: Small reorganization and tidier formatting for the main help text printed by `terraform` with no subcommands. [GH-26695]
* cli: Removed the `terraform debug` container command, which has not had any subcommands under it for a long time. [GH-26695]

BUG FIXES:

* cli: Exit with an error if unable to gather input from the UI. For example, this may happen when running in a non-interactive environment but without `-input=false`. Previously Terraform would interpret these errors as empty strings, which could be confusing. [GH-26509]

BREAKING CHANGES:
* backend/atlas: the `atlas` backend, which was deprecated in v0.12, has been removed. [GH-26651]

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
