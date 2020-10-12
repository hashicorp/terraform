## 0.14.0 (Unreleased)

NEW FEATURES:
* `terraform init`: Terraform will now generate a lock file in the configuration directory which you can check in to your version control so that Terraform can make the same version selections in future. [GH-26524]

    If you wish to retain the previous behavior of always taking the newest version allowed by the version constraints on each install, you can run `terraform init -upgrade` to see that behavior.

UPGRADE NOTES:
* configs: The `version` argument inside provider configuration blocks has been documented as deprecated since Terraform 0.12. As of 0.14 it will now also generate an explicit deprecation warning. To avoid the warning, use [provider requirements](https://www.terraform.io/docs/configuration/provider-requirements.html) declarations instead. ([#26135](https://github.com/hashicorp/terraform/issues/26135))
* The official MacOS builds of Terraform now require MacOS 10.12 Sierra or later. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* TLS certificate verification for outbound HTTPS requests from Terraform CLI no longer treats the certificate's "common name" as a valid hostname when the certificate lacks any "subject alternative name" entries for the hostname. TLS server certificates must list their hostnames as a "DNS name" in the subject alternative names field. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* Outbound HTTPS requests from Terraform CLI now enforce [RFC 8446](https://tools.ietf.org/html/rfc8446)'s client-side downgrade protection checks. This should not significantly affect normal operation, but may result in connection errors in environments where outgoing requests are forced through proxy servers and other "middleboxes", if they have behavior that resembles a downgrade attack. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* Terraform's HTTP client code is now slightly stricter than before in HTTP header parsing, but in ways that should not affect typical server implementations: Terraform now trims only _ASCII_ whitespace characters, and does not allow `Transfer-Encoding: identity`. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* The `terraform 0.13upgrade` subcommand and the associated upgrade mechanisms are no longer available. Complete the v0.13 upgrade process before upgrading to Terraform v0.14.

ENHANCEMENTS:

* `terraform plan` and `terraform apply`: Added an experimental concise diff renderer. By default, Terraform plans now hide most unchanged fields, only displaying the most relevant changes and some identifying context. This experiment can be disabled by setting a `TF_X_CONCISE_DIFF` environment variable to `0`. ([#26187](https://github.com/hashicorp/terraform/issues/26187))
* cli: A new global command line option `-chdir=...`, placed before the selected subcommand, instructs Terraform to switch to a different working directory before executing the subcommand. This is similar to switching to a new directory with `cd` before running Terraform, but it avoids changing the state of the calling shell. ([#26087](https://github.com/hashicorp/terraform/issues/26087))
* configs: Added `sensitive` argument for variable blocks, which supresses output where that variable is used ([#26183](https://github.com/hashicorp/terraform/pull/26183))
* configs: Added `alltrue` function, which returns `true` if all elements in the given collection are `true`. This is primarily intended to make it easier to write variable validation conditions which operate on collections. ([#25656](https://github.com/hashicorp/terraform/issues/25656))
* core: `terraform plan` no longer uses a separate refresh phase, all resources are updated on-demand during planning ([#26270](https://github.com/hashicorp/terraform/issues/26270))
* core: `ignore_changes` can now apply to map keys that are not listed in the configuration ([#26421](https://github.com/hashicorp/terraform/issues/26421))
* `terraform console`: Now has distinct rendering of lists, sets, and tuples, and correctly renders objects with `null` attribute values. ([#26189](https://github.com/hashicorp/terraform/issues/26189))
* `terraform login`: Added support for OAuth2 application scopes. ([#26239](https://github.com/hashicorp/terraform/issues/26239))
* `terraform fmt`: Will now do some slightly more opinionated normalization behaviors, using the documented idiomatic syntax. ([#26390](https://github.com/hashicorp/terraform/issues/26390))
* `terraform init`'s provider installation step will now abort promptly if Terraform receives an interrupt signal. ([#26405](https://github.com/hashicorp/terraform/issues/26405))
* backend/consul: Split state into chunks when outgrowing the limit of the Consul KV store. This allows storing state larger than the Consul 512KB limit. ([#25856](https://github.com/hashicorp/terraform/issues/25856))
* backend/consul: Add force-unlock support to the Consul backend ([#25837](https://github.com/hashicorp/terraform/issues/25837))
* On Unix-based operating systems other than MacOS, the `SSL_CERT_DIR` environment variable can now be a colon-separated list of multiple certificate search paths. ([#26357](https://github.com/hashicorp/terraform/issues/26357))
* On MacOS, Terraform will now use the `Security.framework` API to access the system trust roots, for improved consistency with other MacOS software. ([#26357](https://github.com/hashicorp/terraform/issues/26357))

BUG FIXES:

* backend/consul: Fix bug which prevented state locking when path has trailing `/` ([#25842](https://github.com/hashicorp/terraform/issues/25842))
* backend/pg: Always have the default workspace in the pg backend ([#26420](https://github.com/hashicorp/terraform/pull/26420))
* backend/pg: Properly quote schema_name in the pg backend configuration ([#26476](https://github.com/hashicorp/terraform/issues/26476))
* build: Fix crash with terraform binary on OpenBSD. ([#26249](https://github.com/hashicorp/terraform/issues/26249)
* command/clistate: return an error on a state unlock failure [[#25729](https://github.com/hashicorp/terraform/issues/25729)] 
* command/format: Fix incorrect heredoc syntax in plan diff output ([#25725](https://github.com/hashicorp/terraform/issues/25725))
* command/taint: If the configuration's `required_version` constraint is not met, the `taint` subcommand will now correctly exit early. ([#26345](https://github.com/hashicorp/terraform/issues/26345))
* command/taint, untaint: Fix issue when using `taint` (and `untaint`) with workspaces where statefile was not found. ([#22467](https://github.com/hashicorp/terraform/issues/22467))
* configs: Report an error when provider configuration attributes are incorrectly added to a `required_providers` object. ([#26184](https://github.com/hashicorp/terraform/issues/26184))
* configs: Better errors for invalid terraform version constraints [GH-26543]
* core: Errors with data sources reading old data during refresh, failing to refresh, and not appearing to wait on resource dependencies are fixed by updates to the data source lifecycle and the merging of refresh and plan ([#26270](https://github.com/hashicorp/terraform/issues/26270))
* core: Prevent evaluation of deposed instances, which in turn prevents errors when referencing create_before_destroy resources that have changes to their count or for_each values ([#25631](https://github.com/hashicorp/terraform/issues/25631))
* lang/funcs: fix panic when `element()` is called with a negative offset ([#26079](https://github.com/hashicorp/terraform/issues/26079))
* lang/funcs: `lookup()` will now only treat map as unknown if it is wholly unknown ([#26427](https://github.com/hashicorp/terraform/issues/26427))
* states/remote: fix `state push -force` to work for all backends ([#26190](https://github.com/hashicorp/terraform/issues/26190))

EXPERIMENTS:

* `module_variable_optional_attrs`: When declaring an input variable for a module whose type constraint (`type` argument) contains an object type constraint, the type expressions for the attributes can be annotated with the experimental `optional(...)` modifier.

    Marking an attribute as "optional" changes the type conversion behavior for that type constraint so that if the given value is a map or object that has no attribute of that name then Terraform will silently give that attribute the value `null`, rather than returning an error saying that it is required. The resulting value still conforms to the type constraint in that the attribute is considered to be present, but references to it in the recieving module will find a null value and can act on that accordingly.
    
    If you try this feature during its experimental period and have feedback about it, please open a feature request issue. We are aiming to stabilize this feature in the forthcoming 0.15 release, but its design may change in the meantime based on feedback. If we make further changes to the feature during the 0.15 period then they will be reflected in 0.15 alpha releases.

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
