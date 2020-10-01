## 0.14.0 (Unreleased)

UPGRADE NOTES:
* configs: The `version` argument inside provider configuration blocks has been documented as deprecated since Terraform 0.12. As of 0.14 it will now also generate an explicit deprecation warning. To avoid the warning, use [provider requirements](https://www.terraform.io/docs/configuration/provider-requirements.html) declarations instead. ([#26135](https://github.com/hashicorp/terraform/issues/26135))
* The official MacOS builds of Terraform now require MacOS 10.12 Sierra or later. [GH-26357]
* TLS certificate verification for outbound HTTPS requests from Terraform CLI no longer treats the certificate's "common name" as a valid hostname when the certificate lacks any "subject alternative name" entries for the hostname. TLS server certificates must list their hostnames as a "DNS name" in the subject alternative names field. [GH-26357]
* Outbound HTTPS requests from Terraform CLI now enforce [RFC 8446](https://tools.ietf.org/html/rfc8446)'s client-side downgrade protection checks. This should not significantly affect normal operation, but may result in connection errors in environments where outgoing requests are forced through proxy servers and other "middleboxes", if they have behavior that resembles a downgrade attack. [GH-26357]
* Terraform's HTTP client code is now slightly stricter than before in HTTP header parsing, but in ways that should not affect typical server implementations: Terraform now trims only _ASCII_ whitespace characters, and does not allow `Transfer-Encoding: identity`. [GH-26357]
* The `terraform 0.13upgrade` subcommand and the associated upgrade mechanisms are no longer available. Complete the v0.13 upgrade process before upgrading to Terraform v0.14.

ENHANCEMENTS:

* `terraform plan` and `terraform apply`: Added an experimental concise diff renderer. By default, Terraform plans now hide most unchanged fields, only displaying the most relevant changes and some identifying context. This experiment can be disabled by setting a `TF_X_CONCISE_DIFF` environment variable to `0`. ([#26187](https://github.com/hashicorp/terraform/issues/26187))
* cli: A new global command line option `-chdir=...`, placed before the selected subcommand, instructs Terraform to switch to a different working directory before executing the subcommand. This is similar to switching to a new directory with `cd` before running Terraform, but it avoids changing the state of the calling shell. ([#26087](https://github.com/hashicorp/terraform/issues/26087))
* configs: Added `sensitive` argument for variable blocks, which supresses output where that variable is used ([#26183](https://github.com/hashicorp/terraform/pull/26183))
* configs: Added `alltrue` function, which returns `true` if all elements in the given collection are `true`. This is primarily intended to make it easier to write variable validation conditions which operate on collections. ([#25656](https://github.com/hashicorp/terraform/issues/25656))
* core: `terraform plan` no longer uses a separate refresh phase, all resources are updated on-demand during planning ([#26270](https://github.com/hashicorp/terraform/issues/26270))
* `terraform console`: Now has distinct rendering of lists, sets, and tuples, and correctly renders objects with `null` attribute values. ([#26189](https://github.com/hashicorp/terraform/issues/26189))
* `terraform login`: Added support for OAuth2 application scopes. ([#26239](https://github.com/hashicorp/terraform/issues/26239))
* `terraform fmt`: Will now do some slightly more opinionated normalization behaviors, using the documented idiomatic syntax. [GH-26390]
* `terraform init`'s provider installation step will now abort promptly if Terraform receives an interrupt signal. [GH-26405]
* backend/consul: Split state into chunks when outgrowing the limit of the Consul KV store. This allows storing state larger than the Consul 512KB limit. ([#25856](https://github.com/hashicorp/terraform/issues/25856))
* On Unix-based operating systems other than MacOS, the `SSL_CERT_DIR` environment variable can now be a colon-separated list of multiple certificate search paths. [GH-26357]
* On MacOS, Terraform will now use the `Security.framework` API to access the system trust roots, for improved consistency with other MacOS software. [GH-26357]

BUG FIXES:

* backend/consul: Fix bug which prevented state locking when path has trailing `/` ([#25842](https://github.com/hashicorp/terraform/issues/25842))
* backend/pg: Always have the default workspace in the pg backend ([#26420](https://github.com/hashicorp/terraform/pull/26420))
* build: Fix crash with terraform binary on OpenBSD. ([#26249](https://github.com/hashicorp/terraform/issues/26249)
* command/clistate: return an error on a state unlock failure [[#25729](https://github.com/hashicorp/terraform/issues/25729)] 
* command/taint: If the configuration's `required_version` constraint is not met, the `taint` subcommand will now correctly exit early. [GH-26345]
* command/taint, untaint: Fix issue when using `taint` (and `untaint`) with workspaces where statefile was not found. [GH-22467]
* configs: Report an error when provider configuration attributes are incorrectly added to a `required_providers` object. ([#26184](https://github.com/hashicorp/terraform/issues/26184))
* core: Errors with data sources reading old data during refresh, failing to refresh, and not appearing to wait on resource dependencies are fixed by updates to the data source lifecycle and the merging of refresh and plan ([#26270](https://github.com/hashicorp/terraform/issues/26270))
* lang/funcs: fix panic when element() is called with a negative offset ([#26079](https://github.com/hashicorp/terraform/issues/26079))
* states/remote: fix `state push -force` to work for all backends ([#26190](https://github.com/hashicorp/terraform/issues/26190))


## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
