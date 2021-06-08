Development of the Terraform v0.15 series continued as [the Terraform v1.0 series](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md). We don't intend to publish any new Terraform v0.15 releases and we recommend that everyone should upgrade from Terraform v0.15.5 to the latest available release in the v1.0 series.

## 0.15.5 (June 02, 2021)

BUG FIXES:

* `terraform plan` and `terraform apply`: Don't show "Objects have changed" notification when the detected changes are only internal details related to legacy SDK quirks. ([#28796](https://github.com/hashicorp/terraform/issues/28796))
* core: Prevent crash during planning when encountering a deposed instance that has been removed from the configuration. ([#28766](https://github.com/hashicorp/terraform/issues/28766))
* core: Fix crash when rendering changes to deposed instances outside of Terraform. ([#28796](https://github.com/hashicorp/terraform/issues/28796))
* core: Restore a missing error when attempting to import a non-existent remote object. ([#28808](https://github.com/hashicorp/terraform/issues/28808))
* core: Fix bug where Terraform failed to release the state lock when applying a stale saved plan failed. ([#28819](https://github.com/hashicorp/terraform/issues/28819))

## 0.15.4 (May 19, 2021)

NEW FEATURES:

* **Noting changes made outside of Terraform:** Terraform has always, by default, made a point during the planning operation of reading the current state of remote objects in order to detect any changes made outside of Terraform, to make sure the plan will take those into account.

    Terraform will now report those detected changes as part of the plan result, in order to give additional context about the planned changes. We've often heard that people find it confusing when a plan includes a change that doesn't seem to be prompted by any recent change in the _configuration_, and so this feature is aiming to provide the previously-missing explanation for situations where Terraform is planning to undo a change.
    
    It can also be useful just as general information when the change _won't_ be undone by Terraform: if you've intentionally made a change outside of Terraform and mirrored that change in your configuration then Terraform will now confirm that it noticed the change you made and took it into account when planning.
    
    By default this new output is for information only and doesn't change any behavior. If Terraform detects a change you were _expecting_ then you don't need to take any additional action to respond to it. However, we've also added a new planning mode `-refresh-only` which allows you to explicitly plan and apply the action of writing those detected changes to the Terraform state, which serves as a plannable replacement for `terraform refresh`. We don't have any plans to remove the long-standing `terraform refresh` command, but we do recommend using `terraform apply -refresh-only` instead in most cases, because it will provide an opportunity to review what Terraform detected before updating the Terraform state.

UPGRADE NOTES:

* This release adds some new reserved reference prefixes to make them available for later work. These are `resource.`, `template.`, `arg.`, and `lazy.`. We don't expect these additions to cause problems for most existing configurations, but could cause a conflict if you are using a custom provider which has a resource type named exactly "resource", "template", "arg", or "lazy". In that unlikely event, you can escape references to resources of those types by adding a `resource.` prefix; for example, if you have a `resource "template" "foo"` then you can change references to it from `template.foo` to `resource.template.foo` in order to escape the new meaning.

ENHANCEMENTS:

* config: The various functions that compute hashs of files on disk, like `filesha256`, will now stream the contents of the given file into the hash function in smaller chunks. Previously they would always read the entire file into memory before hashing it, due to following a similar implementation strategy as the `file` function. ([#28681](https://github.com/hashicorp/terraform/issues/28681))
* config: Some new escaping syntax which is not yet useful but will be part of the backward-compatibility story for certain future language editions. ([#28709](https://github.com/hashicorp/terraform/issues/28709))
* core: Resource diagnostics are no longer lost on remote state storage failures ([#28724](https://github.com/hashicorp/terraform/issues/28724))
* core: Diagnostics from provisioner failures are now shown in CLI output ([#28753](https://github.com/hashicorp/terraform/issues/28753))
* `terraform init`: add a new `-migrate-state` flag instead of automatic state migration, to prevent failing when old backend config is not usable ([#28718](https://github.com/hashicorp/terraform/issues/28718))
* `terraform plan` and `terraform apply`: will now report any changes Terraform detects during the "refresh" phase for each managed object, providing confirmation that Terraform has seen those changes and, where appropriate, extra context to help understand the planned change actions that follow. ([#28634](https://github.com/hashicorp/terraform/issues/28634))
* `terraform plan` and `terraform apply`: now have a new option `-refresh-only` to activate the "refresh only" planning mode, which causes Terraform to ignore any changes suggested by the configuration but still detect any changes made outside of Terraform since the latest `terraform apply`. ([#28634](https://github.com/hashicorp/terraform/issues/28634))
* backend/gcs: Terraform Core now supports [Workload Identity Federation](https://cloud.google.com/iam/docs/workload-identity-federation). The federated JSON credentials must be loaded through the `GOOGLE_APPLICATION_CREDENTIALS` environment variable. This is also available in the Google Provider in versions newer than v3.61. ([#28296](https://github.com/hashicorp/terraform/issues/28296))
* backend/remote: supports several new CLI options when running plans and applies with Terraform Cloud: `-refresh=false`, `-replace`, and `-refresh-only`. ([#28746](https://github.com/hashicorp/terraform/issues/28746))

BUG FIXES:

* core: Fix sensitivity handling with plan values, which could cause the sensitive marks to be lost during apply leading to a perpetual diff ([#28687](https://github.com/hashicorp/terraform/issues/28687))
* core: Fix crash when specifying SSH `bastion_port` in a resource `connection` block ([#28665](https://github.com/hashicorp/terraform/issues/28665))
* core: Terraform will now upgrade and refresh (unless disabled) deposed objects during planning, in a similar manner as for objects that have been removed from the configuration. "Deposed" is how Terraform represents the situation where a `create_before_destroy` replacement failed to destroy the old object, in which case Terraform needs to track both the new and old objects until the old object is successfully deleted. Refreshing these during planning means that you can, if you wish, delete a "deposed" object manually outside of Terraform and then have Terraform detect that you've done so. ([#28634](https://github.com/hashicorp/terraform/issues/28634))
* config: Improve the sensitivity support for `lookup` and `length` functions, which were accidentally omitted from the larger update in 0.15.1 ([#28509](https://github.com/hashicorp/terraform/issues/28509))
* backend/gcs: Fixed a bug where service account impersonation didn't work if the original identity was another service account ([#28139](https://github.com/hashicorp/terraform/issues/28139))

## 0.15.3 (May 06, 2021)

ENHANCEMENTS:

* `terraform show`: Add data to the JSON plan output describing which changes caused a resource to be replaced ([#28608](https://github.com/hashicorp/terraform/issues/28608))

BUG FIXES:

* `terraform show`: Fix crash for JSON plan output of new resources with sensitive attributes in nested blocks ([#28624](https://github.com/hashicorp/terraform/issues/28624))

## 0.15.2 (May 05, 2021)

ENHANCEMENTS:

* `terraform plan` and `terraform apply`: Both now support a new planning option `-replace=...` which takes the address of a resource instance already tracked in the state and forces Terraform to upgrade either an update or no-op plan for that instance into a "replace" (either destroy-then-create or create-then-destroy depending on configuration), to allow replacing a degraded object with a new object of the same configuration in a single action and preview the effect of that before applying it.
* `terraform apply`: Now has a `-destroy` option for symmetry with `terraform plan -destroy`, which makes `terraform destroy` effectively an alias for `terraform apply -destroy`. This change is only for consistency between `terraform plan` and `terraform apply`; there are no current plans to deprecate `terraform destroy`. ([#28489](https://github.com/hashicorp/terraform/issues/28489))
* core: Update HCL to allow better planning of dynamic blocks ([#28424](https://github.com/hashicorp/terraform/issues/28424))
* core: Unmark values when planning data sources ([#28539](https://github.com/hashicorp/terraform/issues/28539))

BUG FIXES:

* command/format: Fix various issues with nested-type attribute formatting ([#28600](https://github.com/hashicorp/terraform/issues/28600))
* core: Fix JSON plan output to add sensitivity data for provider-specified sensitive attribute values ([#28523](https://github.com/hashicorp/terraform/issues/28523))
* cli: Fix missing "forces replacement" UI for attribute changes which are marked as sensitive by the provider ([#28583](https://github.com/hashicorp/terraform/issues/28583))
* cli: Fix crash when rendering diagnostic caused by missing trailing quote ([#28598](https://github.com/hashicorp/terraform/issues/28598))
* config: Fix crash when calling `setproduct` with one or more empty collections ([#28607](https://github.com/hashicorp/terraform/issues/28607))

## 0.15.1 (April 26, 2021)

ENHANCEMENTS:

* config: Various Terraform language functions now have more precise inference rules for propagating the "sensitive" characteristic values.
 
    The affected functions are `chunklist`, `concat`, `flatten`, `keys`, `length`, `lookup`, `merge`, `setproduct`, `tolist`, `tomap`, `values`, and `zipmap`. The details are a little different for each of these but the general idea is to, as far as possible, preserve the sensitive characteristic on individual element or attribute values in result structures rather than always conservatively applying sensitivity to the whole result.
    
    The primary benefit of these improvements is that you can now use these functions as part of constructing maps for `for_each` in situations where the input collection is never sensitive but some of the elements/attributes inside might be. ([#28446](https://github.com/hashicorp/terraform/issues/28446)] [[#28460](https://github.com/hashicorp/terraform/issues/28460))
* cli: Update the HashiCorp public key ([#28505](https://github.com/hashicorp/terraform/issues/28505))
* cli: Diagnostic messages can now be annotated with resource and provider addresses. ([#28275](https://github.com/hashicorp/terraform/issues/28275))
* cli: `terraform login` now has a new user experience for successful log-ins to Terraform Cloud and Terraform Enterprise. ([#28487](https://github.com/hashicorp/terraform/issues/28487))
* core: Minor graph performance optimizations. ([#28329](https://github.com/hashicorp/terraform/issues/28329))

BUG FIXES:

* config: Fix validation error when passing providers from a non-default namespace into modules. ([#28414](https://github.com/hashicorp/terraform/issues/28414))
* cli: Fix missing colors and extraneous resource summary for plan/apply with the remote backend. ([#28409](https://github.com/hashicorp/terraform/issues/28409))
* cli: Diagnostics messages will only indicate that a referenced value is sensitive if that value is _directly_ sensitive, as opposed to being a complex-typed value that _contains_ a sensitive value. ([#28442](https://github.com/hashicorp/terraform/issues/28442))
* core: Don't trigger data source reads from changes in sibling module instances. ([#28267](https://github.com/hashicorp/terraform/issues/28267))
* core: Restore saved dependencies when a resource destroy operation fails. ([#28317](https://github.com/hashicorp/terraform/issues/28317))
* core: Fix crash when setting sensitive attributes to a sensitive value. ([#28383](https://github.com/hashicorp/terraform/issues/28383))
* core: Loosen output value sensitivity requirement for non-root modules. This means that modules which may receive sensitive values as input variables no longer need to mark all related outputs as sensitive. The requirement for root modules to specify the `sensitive` attribute for sensitive values remains, with an extended diagnostic message to explain why. ([#28472](https://github.com/hashicorp/terraform/issues/28472))
* provisioner: Fix panic with unexpected null values in provisioner configuration ([#28457](https://github.com/hashicorp/terraform/issues/28457))

## 0.15.0 (April 14, 2021)

UPGRADE NOTES AND BREAKING CHANGES:

The following is a summary of each of the changes in this release that might require special consideration when upgrading. Refer to [the Terraform v0.15 upgrade guide](https://www.terraform.io/upgrade-guides/0-15.html) for more details and recommended upgrade steps.

* "Proxy configuration blocks" (provider blocks with only `alias` set) in shared modules are now replaced with a more explicit `configuration_aliases` argument within the `required_providers` block. Some support for the old syntax is retained for backward compatibility, but we've added explicit error messages for situations where Terraform would previously silently misinterpret the purpose of an empty `provider` block. ([#27739](https://github.com/hashicorp/terraform/issues/27739))

* The `list` and `map` functions, both of which were deprecated since Terraform v0.12, are now removed. You can replace uses of these functions with `tolist([...])` and `tomap({...})` respectively. ([#26818](https://github.com/hashicorp/terraform/issues/26818))

* Terraform now requires UTF-8 character encoding and virtual terminal support when running on Windows. This unifies Terraform's terminal handling on Windows with that of other platforms, as per [Microsoft recommendations](https://docs.microsoft.com/en-us/windows/console/classic-vs-vt). Terraform previously required these terminal features on all other platforms, and now requires them on Windows too.
    
    UTF-8 and virtual terminal support were introduced across various Windows 10 updates, and so Terraform is no longer officially supported on the original release of Windows 10 or on Windows 8 and earlier. However, there are currently no technical measures to artificially _prevent_ Terraform from running on these obsolete Windows releases, and so you _may_ still be able to use Terraform v0.15 on older Windows versions if you either disable formatting (using the `-no-color`) option, or if you use a third-party terminal emulator package such as [ConEmu](https://conemu.github.io/), [Cmder](https://cmder.net/), or [mintty](https://mintty.github.io/).
    
    We strongly encourage planning to migrate to a newer version of Windows rather than relying on these workarounds for the long term, because the Terraform team will test future releases only on up-to-date Windows 10 and can therefore not guarantee ongoing support for older versions.

* Built-in vendor provisioners (chef, habitat, puppet, and salt-masterless) have been removed. ([#26938](https://github.com/hashicorp/terraform/pull/26938))

* Interrupting execution will now cause terraform to exit with a non-zero exit status. ([#26738](https://github.com/hashicorp/terraform/issues/26738))

* The trailing `[DIR]` argument to specify the working directory for various commands is no longer supported. Use the global `-chdir` option instead. ([#27664](https://github.com/hashicorp/terraform/pull/27664))

    For example, instead of `terraform init infra`, write `terraform -chdir=infra init`.
* The `-lock` and `-lock-timeout` options are no longer available on `terraform init` ([#27464](https://github.com/hashicorp/terraform/issues/27464))

* The `-verify-plugins=false` option is no longer available on `terraform init`. (Terraform now _always_ verifies plugins.) ([#27461](https://github.com/hashicorp/terraform/issues/27461))

* The `-get-plugins=false` option is no longer available on `terraform init`. (Terraform now _always_ installs plugins.) ([#27463](https://github.com/hashicorp/terraform/issues/27463))

* The `-force` option is no longer available on `terraform destroy`. Use `-auto-approve` instead ([#27681](https://github.com/hashicorp/terraform/pull/27681))

* The `-var` and `-var-file` options are no longer available on `terraform validate`. These were deprecated and have had no effect since Terraform v0.12. ([#27906](https://github.com/hashicorp/terraform/issues/27906))

* `terraform version -json` output no longer includes the (previously-unpopulated) "revision" property ([#27484](https://github.com/hashicorp/terraform/issues/27484))

* In the `gcs` backend the `path` config argument, which was deprecated since Terraform v0.11, is now removed. Use the `prefix` argument instead. ([#26841](https://github.com/hashicorp/terraform/issues/26841))

* The deprecated `ignore_changes = ["*"]` wildcard syntax is no longer supported. Use `ignore_changes = all` instead. ([#27834](https://github.com/hashicorp/terraform/issues/27834))

* Previously deprecated quoted variable type constraints are no longer supported. Follow the instructions in the error message to update your type signatures to be more explicit. For example, use `map(string)` instead of `"map"`. ([#27852](https://github.com/hashicorp/terraform/issues/27852))

* Terraform will no longer make use of the `HTTP_PROXY` environment variable to determine proxy settings for connecting to HTTPS servers. You must always set `HTTPS_PROXY` if you intend to use a proxy to connect to an HTTPS server. (Note: This affects only connections made directly from Terraform CLI. Terraform providers are separate programs that make their own requests and may thus have different proxy configuration behaviors.)

* Provider-defined sensitive attributes will now be redacted throughout the plan output. You may now see values redacted as `(sensitive)` that were previously visible, because sensitivity did not follow provider-defined sensitive attributes.

    If you are transforming a value and wish to force it _not_ to be sensitive, such as if you are transforming a value in such a way that removes the sensitive data, we recommend using the new `nonsensitive` function to hint Terraform that the result is not sensitive.

* The `atlas` backend, which was deprecated since Terraform v0.12, is now removed. ([#26651](https://github.com/hashicorp/terraform/issues/26651))

* We've upgraded the underlying TLS and certificate-related libraries that Terraform uses when making HTTPS requests to remote systems. This includes the usual tweaks to preferences for different cryptographic algorithms during handshakes and also some slightly-stricter checking of certificate syntax. These changes should not cause problems for correctly-implemented HTTPS servers, but can sometimes cause unexpected behavior changes with servers or middleboxes that don't comply fully with the relevant specifications.

ENHANCEMENTS:

* config: A `required_providers` entry can now contain `configuration_aliases` to declare additional configuration aliases names without requirring a configuration block ([#27739](https://github.com/hashicorp/terraform/issues/27739))
* config: Improved type inference for conditional expressions. ([#28116](https://github.com/hashicorp/terraform/issues/28116))
* config: Provider-defined sensitive attributes will now be redacted throughout the plan output. ([#28036](https://github.com/hashicorp/terraform/issues/28036))
* config: New function `one` for concisely converting a zero-or-one element list/set into a single value that might be `null`. ([#27454](https://github.com/hashicorp/terraform/issues/27454))
* config: New functions `sensitive` and `nonsensitive` allow module authors to explicitly override Terraform's default infererence of value sensitivity for situations where it's too conservative or not conservative enough. ([#27341](https://github.com/hashicorp/terraform/issues/27341))
* config: Terraform will now emit a warning if you declare a `backend` block in a non-root module. Terraform has always ignored such declarations, but previously did so silently. This is a warning rather than an error only because it is sometimes convenient to temporarily use a root module as if it were a child module in order to test or debug its behavior separately from its main backend. ([#26954](https://github.com/hashicorp/terraform/issues/26954))
* config: Removed warning about interpolation-only expressions being deprecated, because `terraform fmt` now automatically fixes most cases that the warning would previously highlight. We still recommend using simpler expressions where possible, but the deprecation warning had caused a common confusion in the community that the interpolation syntax is _always_ deprecated, rather than only in the interpolation-only case. ([#27835](https://github.com/hashicorp/terraform/issues/27835))
* config: The family of error messages with the summary "Invalid for_each argument" will now include some additional context about which external values contributed to the result, making it easier to find the root cause of the error. ([#26747](https://github.com/hashicorp/terraform/issues/26747))
* config: Terraform now does text processing using the rules and tables defined for Unicode 13. Previous versions were using Unicode 12 rules.
* `terraform init`: Will now make suggestions for possible providers on some registry failures, and generally remind of `required_providers` on all registry failures. ([#28014](https://github.com/hashicorp/terraform/issues/28014))
* `terraform init`: Provider installation will now only attempt to rewrite `.terraform.lock.hcl` if it would contain new information. ([#28230](https://github.com/hashicorp/terraform/issues/28230))
* `terraform init`: New `-lockfile=readonly` option, which suppresses writing changes to the dependency lock file. Any installed provider packages must already be recorded in the lock file, or initialization will fail. Use this if you are managing the lock file via a separate process and want to avoid adding new checksums for existing dependencies. ([#27630](https://github.com/hashicorp/terraform/issues/27630))
* `terraform show`: Improved performance when rendering large plans as JSON. ([#27998](https://github.com/hashicorp/terraform/issues/27998))
* `terraform validate`: The JSON output now includes a code snippet object for each diagnostic. If present, this object contains an excerpt of the source code which triggered the diagnostic, similar to what Terraform would include in human-oriented diagnostic messages. ([#28057](https://github.com/hashicorp/terraform/issues/28057))
* cli: Terraform now uses UTF-8 and full VT mode even when running on Windows. Previously Terraform was using the "classic" Windows console API, which was far more limited in what formatting sequences it supported and which characters it could render. ([#27487](https://github.com/hashicorp/terraform/issues/27487))
* cli: Improved support for Windows console UI on Windows 10, including bold colors and underline for HCL diagnostics. ([#26588](https://github.com/hashicorp/terraform/issues/26588))
* cli: Diagnostic messages now have a vertical line along their left margin, which we hope will achieve a better visual hierarchy for sighted users and thus make it easier to see where the errors and warnings start and end in relation to other content that might be printed alongside. ([#27343](https://github.com/hashicorp/terraform/issues/27343))
* cli: Typing an invalid top-level command, like `terraform destory` instead of `destroy`, will now print out a specific error message about the command being invalid, rather than just printing out the usual help directory. ([#26967](https://github.com/hashicorp/terraform/issues/26967))
* cli: Plugin crashes will now be reported with more detail, pointing out the plugin name and the method call along with the stack trace ([#26694](https://github.com/hashicorp/terraform/issues/26694))
* cli: Core and Provider logs can now be enabled separately for debugging, using `TF_LOG_CORE` and `TF_LOG_PROVIDER` ([#26685](https://github.com/hashicorp/terraform/issues/26685))
* backend/azurerm: Support for authenticating as AzureAD users/roles. ([#28181](https://github.com/hashicorp/terraform/issues/28181))
* backend/pg: Now allows locking of each workspace separately, whereas before the locks were global across all workspaces. ([#26924](https://github.com/hashicorp/terraform/issues/26924))

BUG FIXES:

* config: Fix multiple upstream crashes with optional attributes and sensitive values. ([#28116](https://github.com/hashicorp/terraform/issues/28116))
* config: Fix various panics in the experimental `defaults` function. ([#27979](https://github.com/hashicorp/terraform/issues/27979), [#28067](https://github.com/hashicorp/terraform/issues/28067))
* config: Fix crash with resources which have sensitive iterable attributes. ([#28245](https://github.com/hashicorp/terraform/issues/28245))
* config: Fix crash when referencing resources with sensitive fields that may be unknown. ([#28180](https://github.com/hashicorp/terraform/issues/28180))
* `terraform validate`: Validation now ignores providers that lack configuration, which is useful for validating modules intended to be called from other modules which therefore don't include their own provider configurations. ([#24896](https://github.com/hashicorp/terraform/issues/24896))
* `terraform fmt`: Fix `fmt` output when unwrapping redundant multi-line string interpolations ([#28202](https://github.com/hashicorp/terraform/issues/28202))
* `terraform console`: expressions using `path` (`path.root`, `path.module`) now return the same result as they would in a configuration ([#27263](https://github.com/hashicorp/terraform/issues/27263))
* `terraform show`: Fix crash when rendering JSON plans containing iterable unknown values. ([#28253](https://github.com/hashicorp/terraform/issues/28253))
* `terraform show`: fix issue with `child_modules` not properly displaying in certain circumstances. ([#27352](https://github.com/hashicorp/terraform/issues/27352))
* `terraform state list`: fix bug where nested modules' resources were missing ([#27268](https://github.com/hashicorp/terraform/issues/27268))
* `terraform state mv`: fix display names in errors and improve error when failing to target a whole resource ([#27482](https://github.com/hashicorp/terraform/issues/27482))
* `terraform taint`: show resource name in -allow-missing warning ([#27501](https://github.com/hashicorp/terraform/issues/27501))
* `terraform untaint`: show resource name in -allow-missing warning ([#27502](https://github.com/hashicorp/terraform/issues/27502))
* cli: All commands will now exit with an error if unable to read input at an interactive prompt. For example, this may happen when running in a non-interactive environment but without `-input=false`. Previously Terraform would behave as if the user entered an empty string, which often led to confusing results. ([#26509](https://github.com/hashicorp/terraform/issues/26509))
* cli: `TF_LOG` levels other than `trace` will now work reliably. ([#26632](https://github.com/hashicorp/terraform/issues/26632))
* core: Fix crash when trying to create a destroy plan with `-refresh=false`. ([#28272](https://github.com/hashicorp/terraform/issues/28272))
* core: Extend the Terraform plan file format to include information about sensitivity and required-replace. This ensures that the output of `terraform show saved.tfplan` matches `terraform plan`, and sensitive values are elided. ([#28201](https://github.com/hashicorp/terraform/issues/28201))
* core: Ensure that stored dependencies are retained when a resource is removed entirely from the configuration, and `create_before_destroy` ordering is preserved. ([#28228](https://github.com/hashicorp/terraform/issues/28228))
* core: Resources removed from the configuration will now be destroyed before their dependencies are updated. ([#28165](https://github.com/hashicorp/terraform/issues/28165))
* core: Refresh data sources while creating a destroy plan, in case their results are important for destroy operations. ([#27408](https://github.com/hashicorp/terraform/issues/27408))
* core: Fix missing deposed object IDs in apply logs ([#27796](https://github.com/hashicorp/terraform/issues/27796))
* backend/azurerm: Fix nil pointer crashes with some state operations. ([#28181](https://github.com/hashicorp/terraform/issues/28181), [#26721](https://github.com/hashicorp/terraform/pull/26721))
* backend/azure: Fix interactions between state reading, state creating, and locking. ([#26561](https://github.com/hashicorp/terraform/issues/26561))

EXPERIMENTS:

* `provider_sensitive_attrs`: This experiment has now concluded, and its functionality is now on by default. If you were previously participating in this experiment then you can remove the experiment opt-in with no other necessary configuration changes.
* There is now a `terraform test` command, which is currently an experimental feature serving as part of [the Module Testing Experiment](https://www.terraform.io/docs/language/modules/testing-experiment.html). 

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
