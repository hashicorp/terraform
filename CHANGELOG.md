## 0.15.0 (Unreleased)

BREAKING CHANGES:

* config: The `list` and `map` functions, both of which were deprecated since Terraform v0.12, are now removed. You can replace uses of these functions with `tolist([...])` and `tomap({...})` respectively. ([#26818](https://github.com/hashicorp/terraform/issues/26818))
* cli: Interrupting execution will now cause terraform to exit with a non-0 status. ([#26738](https://github.com/hashicorp/terraform/issues/26738))
* cli: `-lock` and `lock-timeout` flags removed from init [GH-27464]
* cli: `-verify-plugins` flag removed from init [GH-27461]
* cli: `-get-plugins` flag removed from init [GH-27463]
* cli: `terraform version -json` will no longer return the revision field [GH-27484] 
* backend/atlas: the `atlas` backend, which was deprecated since v0.12, has been removed. ([#26651](https://github.com/hashicorp/terraform/issues/26651))
* backend/gcs: The `path` config argument, which was deprecated since v0.11, has been removed. Use the `prefix` argument instead. ([#26841](https://github.com/hashicorp/terraform/issues/26841))

ENHANCEMENTS:

* config: Terraform will now emit a warning if you declare a `backend` block in a non-root module. Terraform has always ignored such declarations, but previously did so silently. This is a warning rather than an error only because it is sometimes convenient to temporarily use a root module as if it were a child module in order to test or debug its behavior separately from its main backend. ([#26954](https://github.com/hashicorp/terraform/issues/26954))
* cli: Improved support for Windows console UI on Windows 10, including bold colors and underline for HCL diagnostics. ([#26588](https://github.com/hashicorp/terraform/issues/26588))
* cli: The family of error messages with the summary "Invalid for_each argument" will now include some additional context about which external values contributed to the result. ([#26747](https://github.com/hashicorp/terraform/issues/26747))
* cli: Typing an invalid top-level command, like `terraform destory` instead of `destroy`, will now print out a specific error message about the command being invalid, rather than just printing out the usual help directory. ([#26967](https://github.com/hashicorp/terraform/issues/26967))
* cli: Plugin crashes will now be reported with more detail, pointing out the plugin name and the method call along with the stack trace ([#26694](https://github.com/hashicorp/terraform/issues/26694))
* provisioner/remote-exec: Can now run in a mode that expects the remote system to be running Windows and excuting commands using the Windows command interpreter, rather than a Unix-style shell. Specify the `target_platform` as `"windows"` in the `connection` block. ([#26865](https://github.com/hashicorp/terraform/issues/26865))

BUG FIXES:

* cli: Exit with an error if unable to gather input from the UI. For example, this may happen when running in a non-interactive environment but without `-input=false`. Previously Terraform would interpret these errors as empty strings, which could be confusing. ([#26509](https://github.com/hashicorp/terraform/issues/26509))
* cli: TF_LOG levels other than `trace` will now work correctly ([#26632](https://github.com/hashicorp/terraform/issues/26632))
* cli: Core and Provider logs can now be enabled separately for debugging, using `TF_LOG_CORE` and `TF_LOG_PROVIDER` ([#26685](https://github.com/hashicorp/terraform/issues/26685))
* command/console: expressions using `path` (`path.root`, `path.module`) now return the same result as they would in a configuration ([#27263](https://github.com/hashicorp/terraform/issues/27263))
* command/show: fix issue with child_modules not properly displaying in certain circumstances [GH-27352]
* command/state list: fix bug where nested modules' resources were missing from `state list` output ([#27268](https://github.com/hashicorp/terraform/issues/27268))
* command/state mv: fix display names in errors and improve error when failing to target a whole resource [GH-27482]
* core: validate will now ignore providers without configuration ([#24896](https://github.com/hashicorp/terraform/issues/24896))

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
