## 0.14.7 (Unreleased)

## 0.14.6 (February 04, 2021)

ENHANCEMENTS:

* backend/s3: Add support for AWS Single-Sign On (SSO) cached credentials ([#27620](https://github.com/hashicorp/terraform/issues/27620))

BUG FIXES:

* cli: Rerunning `init` will reuse installed providers rather than fetching the provider again ([#27582](https://github.com/hashicorp/terraform/issues/27582))
* config: Fix panic when applying a config using sensitive values in some block sets ([#27635](https://github.com/hashicorp/terraform/issues/27635))
* core: Fix "Invalid planned change" error when planning tainted resource which no longer exists ([#27563](https://github.com/hashicorp/terraform/issues/27563))
* core: Fix panic when refreshing data source which contains sensitive values ([#27567](https://github.com/hashicorp/terraform/issues/27567))
* core: Fix init with broken link in plugin_cache_dir ([#27447](https://github.com/hashicorp/terraform/issues/27447))
* core: Prevent evaluation of removed data source instances during plan ([#27621](https://github.com/hashicorp/terraform/issues/27621))
* core: Don't plan changes for outputs that remain null ([#27512](https://github.com/hashicorp/terraform/issues/27512))

## 0.14.5 (January 20, 2021)

ENHANCEMENTS:

* backend/pg: The Postgres backend now supports the "scram-sha-256" authentication method. ([#26886](https://github.com/hashicorp/terraform/issues/26886))

BUG FIXES:

* cli: Fix formatting of long integers in outputs and console ([#27479](https://github.com/hashicorp/terraform/issues/27479))
* cli: Fix redundant check of remote workspace version for local operations ([#27498](https://github.com/hashicorp/terraform/pull/27498))
* cli: Fix missing check of remote workspace version for state migration ([#27556](https://github.com/hashicorp/terraform/issues/27556))
* cli: Fix world-writable permissions on dependency lock file ([#27205](https://github.com/hashicorp/terraform/issues/27205))

## 0.14.4 (January 06, 2021)

UPGRADE NOTES:

* This release disables the remote Terraform version check feature for plan and apply operations. This fixes an issue with using custom Terraform version bundles in Terraform Enterprise. ([#27319](https://github.com/hashicorp/terraform/issues/27319))

BUG FIXES:

* backend/remote: Disable remote Terraform workspace version check when the remote workspace is in local operations mode ([#27407](https://github.com/hashicorp/terraform/issues/27407))
* core: Fix panic when using `sensitive` values as arguments to data sources ([#27335](https://github.com/hashicorp/terraform/issues/27335))
* core: Fix panic when using `sensitive` values as `count` arguments on validate ([#27410](https://github.com/hashicorp/terraform/issues/27410))
* core: Fix panic when passing `sensitive` values to module input variables which have custom variable validation ([#27412](https://github.com/hashicorp/terraform/issues/27412))
* dependencies: Upgrade HCL to v2.8.2, fixing several bugs with `sensitive` values ([#27420](https://github.com/hashicorp/terraform/issues/27420))

## 0.14.3 (December 17, 2020)

ENHANCEMENTS:

* `terraform output`: Now supports a new "raw" mode, activated by the `-raw` option, for printing out the raw string representation of a particular output value. ([#27212](https://github.com/hashicorp/terraform/issues/27212))

    Only primitive-typed values have a string representation, so this formatting mode is not compatible with complex types. The `-json` mode is still available as a general way to get a machine-readable representation of an output value of any type.
    
* config: `for_each` now allows maps whose _element values_ are sensitive, as long as the element keys and the map itself are not sensitive. ([#27247](https://github.com/hashicorp/terraform/issues/27247))

BUG FIXES:

* config: Fix `anytrue` and `alltrue` functions when called with values which are not known until apply. ([#27240](https://github.com/hashicorp/terraform/issues/27240))
* config: Fix `sum` function when called with values which are not known until apply. Also allows `sum` to cope with numbers too large to represent in float64, along with correctly handling errors around infinite values. ([#27249](https://github.com/hashicorp/terraform/issues/27249))
* config: Fixed panic when referencing sensitive values in resource `count` expressions ([#27238](https://github.com/hashicorp/terraform/issues/27238))
* config: Fix incorrect attributes in diagnostics when validating objects ([#27010](https://github.com/hashicorp/terraform/issues/27010))
* core: Prevent unexpected updates during plan when multiple sensitive values are involved ([#27318](https://github.com/hashicorp/terraform/issues/27318))
* dependencies: Fix several small bugs related to the use of `sensitive` values with expressions and functions.
* lang: Fix panic when calling `coalescelist` with a `null` argument ([#26988](https://github.com/hashicorp/terraform/issues/26988))
* `terraform apply`: `-refresh=false` was skipped when running apply directly ([#27233](https://github.com/hashicorp/terraform/issues/27233))
* `terraform init`: setting `-get-plugins` to `false` will now cause a warning, as this flag has been a no-op since 0.13.0 and usage is better served through using `provider_installation` blocks ([#27092](https://github.com/hashicorp/terraform/issues/27092))
* `terraform init` and other commands which interact with the dependency lock file: These will now generate a normal error message if the lock file is incorrectly a directory, rather than crashing as before. ([#27250](https://github.com/hashicorp/terraform/issues/27250))

## 0.14.2 (December 08, 2020)

BUG FIXES:

* backend/remote: Disable the remote backend version compatibility check for workspaces set to use the "latest" pseudo-version. ([#27199](https://github.com/hashicorp/terraform/issues/27199))
* providers/terraform: Disable the remote backend version compatibility check for the `terraform_remote_state` data source. This check is unnecessary, because the data source is read-only by definition. ([#27197](https://github.com/hashicorp/terraform/issues/27197))

## 0.14.1 (December 08, 2020)

ENHANCEMENTS:

* backend/remote: When using the enhanced remote backend with commands which locally modify state, verify that the local Terraform version and the configured remote workspace Terraform version are compatible. This prevents accidentally upgrading the remote state to an incompatible version. The check is skipped for commands which do not write state, and can also be disabled by the use of a new command-line flag, `-ignore-remote-version`. ([#26947](https://github.com/hashicorp/terraform/issues/26947))

BUG FIXES:

* configs: Fix for errors when using multiple layers of sensitive input variables ([#27095](https://github.com/hashicorp/terraform/issues/27095))
* configs: Fix error when using sensitive input variables in conditionals ([#27107](https://github.com/hashicorp/terraform/issues/27107))
* core: Fix permanent diff when a resource changes only in sensitivity, for example due to changing the sensitivity of a variable or output used as an attribute value. ([#27128](https://github.com/hashicorp/terraform/issues/27128))
* core: Fix issues where `ignore_changes` appears to not work, or causes validation errors with some resources. ([#27141](https://github.com/hashicorp/terraform/issues/27141))
* `terraform fmt`: Fix incorrect formatting with attribute expressions enclosed in parentheses. ([#27040](https://github.com/hashicorp/terraform/issues/27040))

## 0.14.0 (December 02, 2020)

NEW FEATURES:
* Terraform now supports marking input variables as sensitive, and will propagate that sensitivity through expressions that derive from sensitive input variables.

* `terraform init` will now generate a lock file in the configuration directory which you can check in to your version control so that Terraform can make the same version selections in future. ([#26524](https://github.com/hashicorp/terraform/issues/26524))

    If you wish to retain the previous behavior of always taking the newest version allowed by the version constraints on each install, you can run `terraform init -upgrade` to see that behavior.

* Terraform will now support reading and writing all compatible state files, even from future versions of Terraform. This means that users of Terraform 0.14.0 will be able to share state files with future Terraform versions until a new state file format version is needed. We have no plans to change the state file format at this time. ([#26752](https://github.com/hashicorp/terraform/issues/26752))

UPGRADE NOTES:
* Outputs that reference sensitive values (which includes variables marked as sensitive, other module outputs marked as `sensitive`, or attributes a provider defines as `sensitive` if the `provider_sensitive_attrs` experiment is activated) must _also_ be defined as sensitive, or Terraform will error at plan.
* The `version` argument inside provider configuration blocks has been documented as deprecated since Terraform 0.12. As of 0.14 it will now also generate an explicit deprecation warning. To avoid the warning, use [provider requirements](https://www.terraform.io/docs/configuration/provider-requirements.html) declarations instead. ([#26135](https://github.com/hashicorp/terraform/issues/26135))
* The official MacOS builds of Terraform now require MacOS 10.12 Sierra or later. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* TLS certificate verification for outbound HTTPS requests from Terraform CLI no longer treats the certificate's "common name" as a valid hostname when the certificate lacks any "subject alternative name" entries for the hostname. TLS server certificates must list their hostnames as a "DNS name" in the subject alternative names field. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* Outbound HTTPS requests from Terraform CLI now enforce [RFC 8446](https://tools.ietf.org/html/rfc8446)'s client-side downgrade protection checks. This should not significantly affect normal operation, but may result in connection errors in environments where outgoing requests are forced through proxy servers and other "middleboxes", if they have behavior that resembles a downgrade attack. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* Terraform's HTTP client code is now slightly stricter than before in HTTP header parsing, but in ways that should not affect typical server implementations: Terraform now trims only _ASCII_ whitespace characters, and does not allow `Transfer-Encoding: identity`. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* The `terraform 0.13upgrade` subcommand and the associated upgrade mechanisms are no longer available. Complete the v0.13 upgrade process before upgrading to Terraform v0.14.
* The `debug` command, which did not offer additional functionality, has been removed.

ENHANCEMENTS:

* config: Added `sensitive` argument for variable blocks, which supresses output where that variable is used ([#26183](https://github.com/hashicorp/terraform/pull/26183))
* config: Added `alltrue` and `anytrue` functions, which serve as a sort of dynamic version of the `&&` and `||` or operators, respectively. These are intended to allow evaluating boolean conditions, such as in variable `validation` blocks, across all of the items in a collection using `for` expressions. ([#25656](https://github.com/hashicorp/terraform/issues/25656)], [[#26498](https://github.com/hashicorp/terraform/issues/26498))
* config: New functions `textencodebase64` and `textdecodebase64` for encoding text in various character encodings other than UTF-8. ([#25470](https://github.com/hashicorp/terraform/issues/25470))
* `terraform plan` and `terraform apply`: Added an experimental concise diff renderer. By default, Terraform plans now hide most unchanged fields, only displaying the most relevant changes and some identifying context. This experiment can be disabled by setting a `TF_X_CONCISE_DIFF` environment variable to `0`. ([#26187](https://github.com/hashicorp/terraform/issues/26187))
* config: `ignore_changes` can now apply to map keys that are not listed in the configuration ([#26421](https://github.com/hashicorp/terraform/issues/26421))
* `terraform console`: Now has distinct rendering of lists, sets, and tuples, and correctly renders objects with `null` attribute values. Multi-line strings are rendered using the "heredoc" syntax. ([#26189](https://github.com/hashicorp/terraform/issues/26189), [#27054](https://github.com/hashicorp/terraform/issues/27054))
* `terraform login`: Added support for OAuth2 application scopes. ([#26239](https://github.com/hashicorp/terraform/issues/26239))
* `terraform fmt`: Will now do some slightly more opinionated normalization behaviors, using the documented idiomatic syntax. ([#26390](https://github.com/hashicorp/terraform/issues/26390))
* `terraform init`'s provider installation step will now abort promptly if Terraform receives an interrupt signal. ([#26405](https://github.com/hashicorp/terraform/issues/26405))
* cli: A new global command line option `-chdir=...`, placed before the selected subcommand, instructs Terraform to switch to a different working directory before executing the subcommand. This is similar to switching to a new directory with `cd` before running Terraform, but it avoids changing the state of the calling shell. ([#26087](https://github.com/hashicorp/terraform/issues/26087))
* cli: help text is been reorganized to emphasize the main commands and improve consistency ([#26695](https://github.com/hashicorp/terraform/issues/26695))
* cli: Ensure that provider requirements are met by the locked dependencies for every command. This will help catch errors if the configuration has changed since the last run of `terraform init`. ([#26761](https://github.com/hashicorp/terraform/issues/26761))
* core: When sensitive values are used as part of provisioner configuration, logging is disabled to ensure the values are not displayed to the UI ([#26611](https://github.com/hashicorp/terraform/issues/26611))
* core: `terraform plan` no longer uses a separate refresh phase. Instead, all resources are updated on-demand during planning ([#26270](https://github.com/hashicorp/terraform/issues/26270))
* modules: Adds support for loading modules with S3 virtual hosted-style access ([#26914](https://github.com/hashicorp/terraform/issues/26914))
* backend/consul: Split state into chunks when outgrowing the limit of the Consul KV store. This allows storing state larger than the Consul 512KB limit. ([#25856](https://github.com/hashicorp/terraform/issues/25856))
* backend/consul: Add force-unlock support to the Consul backend ([#25837](https://github.com/hashicorp/terraform/issues/25837))
* backend/gcs: Add service account impersonation to GCS backend ([#26837](https://github.com/hashicorp/terraform/issues/26837))
* On Unix-based operating systems other than MacOS, the `SSL_CERT_DIR` environment variable can now be a colon-separated list of multiple certificate search paths. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* On MacOS, Terraform will now use the `Security.framework` API to access the system trust roots, for improved consistency with other MacOS software. ([#26357](https://github.com/hashicorp/terraform/issues/26357))

BUG FIXES:

* config: Report an error when provider configuration attributes are incorrectly added to a `required_providers` object. ([#26184](https://github.com/hashicorp/terraform/issues/26184))
* config: Better errors for invalid terraform version constraints ([#26543](https://github.com/hashicorp/terraform/issues/26543))
* config: fix panic when `element()` is called with a negative offset ([#26079](https://github.com/hashicorp/terraform/issues/26079))
* config: `lookup()` will now only treat map as unknown if it is wholly unknown ([#26427](https://github.com/hashicorp/terraform/issues/26427))
* config: Fix provider detection for resources when local name does not match provider type ([#26871](https://github.com/hashicorp/terraform/issues/26871))
* `terraform fmt`: Fix incorrect heredoc syntax in plan diff output ([#25725](https://github.com/hashicorp/terraform/issues/25725))
* `terraform show`: Hide sensitive outputs from display ([#26740](https://github.com/hashicorp/terraform/issues/26740))
* `terraform taint`: If the configuration's `required_version` constraint is not met, the `taint` subcommand will now correctly exit early. ([#26345](https://github.com/hashicorp/terraform/issues/26345))
* `terraform taint` and `terraform untaint`: Fix issue when using `taint` (and `untaint`) with workspaces where statefile was not found. ([#22467](https://github.com/hashicorp/terraform/issues/22467))
* `terraform init`: Fix locksfile constraint output for versions like "1.2". ([#26637](https://github.com/hashicorp/terraform/issues/26637))
* `terraform init`: Omit duplicate version constraints when installing packages or writing locksfile. ([#26678](https://github.com/hashicorp/terraform/issues/26678))
* cli: return an error on a state unlock failure [[#25729](https://github.com/hashicorp/terraform/issues/25729)] 
* core: Prevent "Inconsistent Plan" errors when using dynamic with a block of TypeSet ([#26638](https://github.com/hashicorp/terraform/issues/26638))
* core: Errors with data sources reading old data during refresh, failing to refresh, and not appearing to wait on resource dependencies are fixed by updates to the data source lifecycle and the merging of refresh and plan ([#26270](https://github.com/hashicorp/terraform/issues/26270))
* core: Prevent evaluation of deposed instances, which in turn prevents errors when referencing create_before_destroy resources that have changes to their count or for_each values ([#25631](https://github.com/hashicorp/terraform/issues/25631))
* core: fix `state push -force` to work for all backends ([#26190](https://github.com/hashicorp/terraform/issues/26190))
* backend/consul: Fix bug which prevented state locking when path has trailing `/` ([#25842](https://github.com/hashicorp/terraform/issues/25842))
* backend/pg: Always have the default workspace in the pg backend ([#26420](https://github.com/hashicorp/terraform/pull/26420))
* backend/pg: Properly quote schema_name in the pg backend configuration ([#26476](https://github.com/hashicorp/terraform/issues/26476))
* build: Fix crash with terraform binary on OpenBSD. ([#26249](https://github.com/hashicorp/terraform/issues/26249))
* internal: Use default AWS credential handling when fetching modules ([#26762](https://github.com/hashicorp/terraform/pull/26762))

EXPERIMENTS:

_Experiments_ are Terraform language features that are not yet finalized but that we've included in a release so you can potentially try them out and share feedback. These features are only available if you explicitly enable the relevant experiment for your module. To share feedback on active experiments, please open an enhancement request issue in the main Terraform repository.

* `module_variable_optional_attrs`: When declaring an input variable for a module whose type constraint (`type` argument) contains an object type constraint, the type expressions for the attributes can be annotated with the experimental `optional(...)` modifier.

    Marking an attribute as "optional" changes the type conversion behavior for that type constraint so that if the given value is a map or object that has no attribute of that name then Terraform will silently give that attribute the value `null`, rather than returning an error saying that it is required. The resulting value still conforms to the type constraint in that the attribute is considered to be present, but references to it in the recieving module will find a null value and can act on that accordingly.
    
    This experiment also includes a function named `defaults` which you can use in a local value to replace the null values representing optional attributes with non-null default values. The function also requires that you enable the `module_variable_optional_attrs` experiment for any module which calls it.
    
* `provider_sensitive_attrs`: This is an unusual experiment in that it doesn't directly allow you to use a new feature in your module configuration but instead it changes the automatic behavior of Terraform in modules where it's enabled.

    For modules where this experiment is active, Terraform will consider the attribute sensitivity flags set in provider resource type schemas when propagating the "sensitive" flag through expressions in the configuration. This is experimental because it has the potential to make far more items in the output be marked as sensitive than before, and so we want to get some experience and feedback about it before hopefully making this the default behavior.
    
    One important consequence of enabling this experiment is that you may need to mark more of your module's output values as `sensitive = true`, in any case where a particular output value is derived from a value a provider has indicated as being sensitive. Without that explicit annotation, Terraform will return an error to avoid implicitly exposing a sensitive value via an output value.

If you try either of these features during their experimental periods and have feedback about them, please open a feature request issue. We are aiming to stabilize both features in the forthcoming v0.15 release, but their design may change in the meantime based on feedback. If we make further changes to the features during the v0.15 period then they will be reflected in v0.15 alpha releases.

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
