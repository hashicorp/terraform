## 0.15.0-beta2 (unreleased)

UPGRADE NOTES:

The output of `terraform validate -json` has been extended to include a code snippet object for each diagnostic. If present, this object contains an excerpt of the source code which triggered the diagnostic. Existing fields in the JSON output remain the same as before. [See the `validate` documentation for more details on the JSON output format](https://www.terraform.io/docs/cli/commands/validate.html). [GH-28057]

ENHANCEMENTS:

* core: Reduce string allocations to improve execution time when rendering large plans as JSON [GH-27998]
* core: provider-defined sensitive attributes redaction is no longer experimental, but default behavior [GH-28036]
* init: Give suggestions for possible providers on some registry failures, and generally remind of `required_providers` on all registry failures [GH-28014]
* init: Add `-lockfile=readonly` flag, which suppresses writing changes to the dependency lock file. Depencies must be able to be verified against the read-only lock file, or initialization will fail. This is useful if you are managing the lock file in a separate process and want to avoid adding new hashes for existing dependencies. [GH-27630]

BUG FIXES:

* Fix for missing configuration snippets in diagnostics, a bug introduced in 0.15.0-beta1 [GH-27944]
* functions: Fix panics in `defaults` caused by missing nested optional collection types, and mismatched primitive fallback types [GH-27979]
* functions: Fix panics in `defaults` caused by missing nested optional structural types, and corresponding missing defaults [GH-28067]

## 0.15.0-beta1 (February 24, 2021)

BREAKING CHANGES:

* Empty provider configuration blocks should be removed from modules. If a configuration alias is required within the module, it can be defined using the `configuration_aliases` argument within `required_providers`. Existing module configurations which were accepted but could produce incorrect or undefined behavior may now return errors when loading the configuration. ([#27739](https://github.com/hashicorp/terraform/issues/27739))
* The `list` and `map` functions, both of which were deprecated since Terraform v0.12, are now removed. You can replace uses of these functions with `tolist([...])` and `tomap({...})` respectively. ([#26818](https://github.com/hashicorp/terraform/issues/26818))
* Terraform now requires UTF-8 character encoding and virtual terminal support when running on Windows. This unifies Terraform's terminal handling on Windows with that of other platforms, as per [Microsoft recommendations](https://docs.microsoft.com/en-us/windows/console/classic-vs-vt). Terraform previously required these terminal features on all other platforms, and now requires them on Windows too.
    
    UTF-8 and virtual terminal support were introduced across various Windows 10 updates, and so Terraform is no longer officially supported on the original release of Windows 10 or on Windows 8 and earlier. However, there are currently no technical measures to artificially _prevent_ Terraform from running on these obsolete Windows releases, and so you _may_ still be able to use Terraform v0.15 on older Windows versions if you either disable formatting (using the `-no-color`) option, or if you use a third-party terminal emulator package such as [ConEmu](https://conemu.github.io/), [Cmder](https://cmder.net/), or [mintty](https://mintty.github.io/).
    
    We strongly encourage planning to migrate to a newer version of Windows rather than relying on these workarounds for the long term, because the Terraform team will test future releases only on up-to-date Windows 10 and can therefore not guarantee ongoing support for older versions.
    
* Interrupting execution will now cause terraform to exit with a non-zero exit status. ([#26738](https://github.com/hashicorp/terraform/issues/26738))
* The trailing `[DIR]` argument to specify the working directory for various commands is no longer supported. Use the global `-chdir` option instead. ([#27664](https://github.com/hashicorp/terraform/pull/27664))

    For example, instead of `terraform init infra`, write `terraform -chdir=infra init`.
* The `-lock` and `-lock-timeout` options are no longer available on `terraform init` ([#27464](https://github.com/hashicorp/terraform/issues/27464))
* The `-verify-plugins=false` option is no longer available on `terraform init`. (Terraform now _always_ verifies plugins.) ([#27461](https://github.com/hashicorp/terraform/issues/27461))
* The `-get-plugins=false` option is no longer available on `terraform init`. (Terraform now _always_ installs plugins.) ([#27463](https://github.com/hashicorp/terraform/issues/27463))
* The `-force` option is no longer available on `terraform destroy`. Use `-auto-approve` instead ([#27681](https://github.com/hashicorp/terraform/pull/27681))
* The `-var` and `-var-file` options are no longer available on `terraform validate`. These had no effect and were deprecated. ([#27906](https://github.com/hashicorp/terraform/issues/27906))
* `terraform version -json` output no longer includes the (previously-unpopulated) "revision" property ([#27484](https://github.com/hashicorp/terraform/issues/27484))
* The `atlas` backend, which was deprecated since Terraform v0.12, is now removed. ([#26651](https://github.com/hashicorp/terraform/issues/26651))
* In the `gcs` backend the `path` config argument, which was deprecated since Terraform v0.11, is now removed. Use the `prefix` argument instead. ([#26841](https://github.com/hashicorp/terraform/issues/26841))
* The deprecated `ignore_changes = ["*"]` wildcard syntax will now error. Use `= all` instead. ([#27834](https://github.com/hashicorp/terraform/issues/27834))
* Previously deprecated quoted type strings will now error rather than warn - follow the instructions in the error message to update your type signatures to be more explicit. For example, use `map(string)` instead of `"map"`. ([#27852](https://github.com/hashicorp/terraform/issues/27852))
* Terraform will no longer make use of the `HTTP_PROXY` environment variable to determine proxy settings for connecting to HTTPS servers. You must always set `HTTPS_PROXY` if you intend to use a proxy to connect to an HTTPS server. (Note: This affects only connections made directly from Terraform CLI. Terraform providers are separate programs that make their own requests and may thus have different proxy configuration behaviors.)
* We've upgraded the underlying TLS and certificate-related libraries that Terraform uses when making HTTPS requests to remote systems. This includes the usual tweaks to preferences for different cryptographic algorithms during handshakes and also some slightly-stricter checking of certificate syntax. These changes should not cause problems for correctly-implemented HTTPS servers, but can sometimes cause unexpected behavior changes with servers or middleboxes that don't comply fully with the relevant specifications.

ENHANCEMENTS:

* backend/azurerm: updating the dependencies for the Azure Backend ([#26721](https://github.com/hashicorp/terraform/pull/26721))
* config: A `required_providers` entry can now contain `configuration_aliases` to declare additional configuration aliases names without requirring a configuration block ([#27739](https://github.com/hashicorp/terraform/issues/27739))
* config: Terraform will now emit a warning if you declare a `backend` block in a non-root module. Terraform has always ignored such declarations, but previously did so silently. This is a warning rather than an error only because it is sometimes convenient to temporarily use a root module as if it were a child module in order to test or debug its behavior separately from its main backend. ([#26954](https://github.com/hashicorp/terraform/issues/26954))
* config: Removed warning surrounding interpolation-only expressions - many of these are caught by `fmt` and we are removing the warning rather than upgrading it to an error ([#27835](https://github.com/hashicorp/terraform/issues/27835))
* config: Terraform now does text processing using the rules and tables defined for Unicode 13. Previous versions were using Unicode 12 rules.
* cli: The family of error messages with the summary "Invalid for_each argument" will now include some additional context about which external values contributed to the result. ([#26747](https://github.com/hashicorp/terraform/issues/26747))
* cli: Terraform now uses UTF-8 and full VT mode even when running on Windows. Previously Terraform was using the "classic" Windows console API, which was far more limited in what formatting sequences it supported and which characters it could render. ([#27487](https://github.com/hashicorp/terraform/issues/27487))
* cli: Improved support for Windows console UI on Windows 10, including bold colors and underline for HCL diagnostics. ([#26588](https://github.com/hashicorp/terraform/issues/26588))
* cli: Diagnostic messages now have a vertical line along their left margin, which we hope will achieve a better visual hierarchy for sighted users and thus make it easier to see where the errors and warnings start and end in relation to other content that might be printed alongside. ([#27343](https://github.com/hashicorp/terraform/issues/27343))
* cli: Typing an invalid top-level command, like `terraform destory` instead of `destroy`, will now print out a specific error message about the command being invalid, rather than just printing out the usual help directory. ([#26967](https://github.com/hashicorp/terraform/issues/26967))
* cli: Plugin crashes will now be reported with more detail, pointing out the plugin name and the method call along with the stack trace ([#26694](https://github.com/hashicorp/terraform/issues/26694))
* cli: Values in files for undeclared variables (ex. `tfvars`) are no longer deprecated, but will continue to produce a warning. The number of warnings produced has been reduced from 3 full warnings before a summary to two. To provide "global" values across configurations, use `TF_VAR...` environment variables. To reduce the verbosity of the warnings, use the existing `-compact-warnings` option. ([#27795](https://github.com/hashicorp/terraform/issues/27795))
* cli: The cli now handles structured logs throughout, allowing for additional log context from providers to be maintained, and offering new options for output filters. ([#26632](https://github.com/hashicorp/terraform/issues/26632))
* cli: Core and Provider logs can now be enabled separately for debugging, using `TF_LOG_CORE` and `TF_LOG_PROVIDER` ([#26685](https://github.com/hashicorp/terraform/issues/26685))
* cli: Experimental `terraform test` command. (TODO: Include a link to the experiment's documentation page as part of aggregating the 0.15.0 prerelease changelogs into the final 0.15.0 changelog) ([#27873](https://github.com/hashicorp/terraform/issues/27873))

BUG FIXES:

* cli: Exit with an error if unable to gather input from the UI. For example, this may happen when running in a non-interactive environment but without `-input=false`. Previously Terraform would interpret these errors as empty strings, which could be confusing. ([#26509](https://github.com/hashicorp/terraform/issues/26509))
* cli: TF_LOG levels other than `trace` will now work correctly ([#26632](https://github.com/hashicorp/terraform/issues/26632))
* command/console: expressions using `path` (`path.root`, `path.module`) now return the same result as they would in a configuration ([#27263](https://github.com/hashicorp/terraform/issues/27263))
* command/show: fix issue with child_modules not properly displaying in certain circumstances ([#27352](https://github.com/hashicorp/terraform/issues/27352))
* command/state list: fix bug where nested modules' resources were missing from `state list` output ([#27268](https://github.com/hashicorp/terraform/issues/27268))
* command/state mv: fix display names in errors and improve error when failing to target a whole resource ([#27482](https://github.com/hashicorp/terraform/issues/27482))
* command/taint: show resource name in -allow-missing warning ([#27501](https://github.com/hashicorp/terraform/issues/27501))
* command/untaint: show resource name in -allow-missing warning ([#27502](https://github.com/hashicorp/terraform/issues/27502))
* core: validate will now ignore providers without configuration ([#24896](https://github.com/hashicorp/terraform/issues/24896))
* core: refresh data sources during destroy ([#27408](https://github.com/hashicorp/terraform/issues/27408))
* core: fix missing deposed object ID in apply logs ([#27796](https://github.com/hashicorp/terraform/issues/27796))
* backend/azure: azure state refreshes outside of grabbing the lock ([#26561](https://github.com/hashicorp/terraform/issues/26561))

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
