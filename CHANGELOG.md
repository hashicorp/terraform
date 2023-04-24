## 1.4.6 (Unreleased)

BUG FIXES

* Fix bug when rendering plans that include null strings. ([#33029](https://github.com/hashicorp/terraform/issues/33029))
* Fix bug when rendering plans that include unknown values in maps. ([#33029](https://github.com/hashicorp/terraform/issues/33029))
* Fix bug where the plan would render twice when using older versions of TFE as a backend. ([#33018](https://github.com/hashicorp/terraform/issues/33018))
* Fix bug where sensitive and unknown metadata was not being propagated to dynamic types while rendering plans. ([#33057](https://github.com/hashicorp/terraform/issues/33057))
* Fix bug where sensitive metadata from the schema was not being included in the `terraform show -json` output. ([#33059](https://github.com/hashicorp/terraform/issues/33059))
* Fix bug where the computed attributes were not being rendered with the `# forces replacement` suffix. ([#33065](https://github.com/hashicorp/terraform/issues/33065))

## 1.4.5 (April 12, 2023)

* Revert change from [[#32892](https://github.com/hashicorp/terraform/issues/32892)] due to an upstream crash.
* Fix planned destroy value which would cause `terraform_data` to fail when being replaced with `create_before_destroy` ([#32988](https://github.com/hashicorp/terraform/issues/32988))

## 1.4.4 (March 30, 2023)

Due to an incident while migrating build systems for the 1.4.3 release where 
`CGO_ENABLED=0` was not set, we are rebuilding that version as 1.4.4 with the 
flag set. No other changes have been made between 1.4.3 and 1.4.4.

## 1.4.3 (March 30, 2023)

BUG FIXES:
* Prevent sensitive values in non-root module outputs from marking the entire output as sensitive ([#32891](https://github.com/hashicorp/terraform/issues/32891))
* Fix the handling of planned data source objects when storing a failed plan ([#32876](https://github.com/hashicorp/terraform/issues/32876))
* Don't fail during plan generation when targeting prevents resources with schema changes from performing a state upgrade ([#32900](https://github.com/hashicorp/terraform/issues/32900))
* Skip planned changes in sensitive marks when the changed attribute is discarded by the provider ([#32892](https://github.com/hashicorp/terraform/issues/32892))

## 1.4.2 (March 16, 2023)

BUG FIXES:

* Fix bug in which certain uses of `setproduct` caused Terraform to crash ([#32860](https://github.com/hashicorp/terraform/issues/32860))
* Fix bug in which some provider plans were not being calculated correctly, leading to an "invalid plan" error ([#32860](https://github.com/hashicorp/terraform/issues/32860))

## 1.4.1 (March 15, 2023)

BUG FIXES:

* Enables overriding modules that have the `depends_on` attribute set, while still preventing the `depends_on` attribute itself from being overridden. ([#32796](https://github.com/hashicorp/terraform/issues/32796))
* `terraform providers mirror`: when a dependency lock file is present, mirror the resolved providers versions, not the latest available based on configuration. ([#32749](https://github.com/hashicorp/terraform/issues/32749))
* Fixed module downloads from S3 URLs when using AWS IAM roles for service accounts (IRSA). ([#32700](https://github.com/hashicorp/terraform/issues/32700))
* hcl: Fix a crash in Terraform when attempting to apply defaults into an incompatible type. ([#32775](https://github.com/hashicorp/terraform/issues/32775))
* Prevent panic when creating a plan which errors before the planning process has begun. ([#32818](https://github.com/hashicorp/terraform/issues/32818))
* Fix the plan renderer skipping the "no changes" messages when there are no-op outputs within the plan. ([#32820](https://github.com/hashicorp/terraform/issues/32820))
* Prevent panic when rendering null nested primitive values in a state output. ([#32840](https://github.com/hashicorp/terraform/issues/32840))
* Warn when an invalid path is specified in `TF_CLI_CONFIG_FILE` ([#32846](https://github.com/hashicorp/terraform/issues/32846))

## 1.4.0 (March 08, 2023)

UPGRADE NOTES:

* config: The `textencodebase64` function when called with encoding "GB18030" will now encode the euro symbol € as the two-byte sequence `0xA2,0xE3`, as required by the GB18030 standard, before applying base64 encoding.
* config: The `textencodebase64` function when called with encoding "GBK" or "CP936" will now encode the euro symbol € as the single byte `0x80` before applying base64 encoding. This matches the behavior of the Windows API when encoding to this Windows-specific character encoding.
* `terraform init`: When interpreting the hostname portion of a provider source address or the address of a module in a module registry, Terraform will now use _non-transitional_ IDNA2008 mapping rules instead of the transitional mapping rules previously used.

    This matches a change to [the WHATWG URL spec's rules for interpreting non-ASCII domain names](https://url.spec.whatwg.org/#concept-domain-to-ascii) which is being gradually adopted by web browsers. Terraform aims to follow the interpretation of hostnames used by web browsers for consistency. For some hostnames containing non-ASCII characters this may cause Terraform to now request a different "punycode" hostname when resolving.
* `terraform init` will now ignore entries in the optional global provider cache directory unless they match a checksum already tracked in the current configuration's dependency lock file. This therefore avoids the long-standing problem that when installing a new provider for the first time from the cache we can't determine the full set of checksums to include in the lock file. Once the lock file has been updated to include a checksum covering the item in the global cache, Terraform will then use the cache entry for subsequent installation of the same provider package. There is an interim CLI configuration opt-out for those who rely on the previous incorrect behavior. ([#32129](https://github.com/hashicorp/terraform/issues/32129))
* The Terraform plan renderer has been completely rewritten to aid with future Terraform Cloud integration. Users should not see any material change in the plan output between 1.3 and 1.4. If you notice any significant differences, or if Terraform fails to plan successfully due to rendering problems, please open a bug report issue.

BUG FIXES:

* The module installer will now record in its manifest a correct module source URL after normalization when the URL given as input contains both a query string portion and a subdirectory portion. Terraform itself doesn't currently make use of this information and so this is just a cosmetic fix to make the recorded metadata more correct. ([#31636](https://github.com/hashicorp/terraform/issues/31636))
* config: The `yamldecode` function now correctly handles entirely-nil YAML documents. Previously it would incorrectly return an unknown value instead of a null value. It will now return a null value as documented. ([#32151](https://github.com/hashicorp/terraform/issues/32151))
* Ensure correct ordering between data sources and the deletion of managed resource dependencies. ([#32209](https://github.com/hashicorp/terraform/issues/32209))
* Fix Terraform creating objects that should not exist in variables that specify default attributes in optional objects. ([#32178](https://github.com/hashicorp/terraform/issues/32178))
* Fix several Terraform crashes that are caused by HCL creating objects that should not exist in variables that specify default attributes in optional objects within collections. ([#32178](https://github.com/hashicorp/terraform/issues/32178))
* Fix inconsistent behaviour in empty vs null collections. ([#32178](https://github.com/hashicorp/terraform/issues/32178))
* `terraform workspace` now returns a non-zero exit when given an invalid argument ([#31318](https://github.com/hashicorp/terraform/issues/31318))
* Terraform would always plan changes when using a nested set attribute ([#32536](https://github.com/hashicorp/terraform/issues/32536))
* Terraform can now better detect when complex optional+computed object attributes are removed from configuration ([#32551](https://github.com/hashicorp/terraform/issues/32551))
* A new methodology for planning set elements can now better detect optional+computed changes within sets ([#32563](https://github.com/hashicorp/terraform/issues/32563))
* Fix state locking and releasing messages when in `-json` mode, messages will now be written in JSON format ([#32451](https://github.com/hashicorp/terraform/issues/32451))
* Fixes a race condition where the Terraform CLI checks if a run is confirmable before the run status gets updated and exits early.

NEW FEATURES:

* When showing the progress of a remote operation running in Terraform Cloud, Terraform CLI will include information about OPA policy evaluation (#32303)

ENHANCEMENTS:

* `terraform plan` can now store a plan file even when encountering errors, which can later be inspected to help identify the source of the failures ([#32395](https://github.com/hashicorp/terraform/issues/32395))
* `terraform_data` is a new builtin managed resource type, which can replace the use of `null_resource`, and can store data of any type ([#31757](https://github.com/hashicorp/terraform/issues/31757))
* `terraform init` will now ignore entries in the optional global provider cache directory unless they match a checksum already tracked in the current configuration's dependency lock file. This therefore avoids the long-standing problem that when installing a new provider for the first time from the cache we can't determine the full set of checksums to include in the lock file. Once the lock file has been updated to include a checksum covering the item in the global cache, Terraform will then use the cache entry for subsequent installation of the same provider package. There is an interim CLI configuration opt-out for those who rely on the previous incorrect behavior. ([#32129](https://github.com/hashicorp/terraform/issues/32129))
* Interactive input for sensitive variables is now masked in the UI ([#29520](https://github.com/hashicorp/terraform/issues/29520))
* A new `-or-create` flag was added to `terraform workspace select`, to aid in creating workspaces in automated situations ([#31633](https://github.com/hashicorp/terraform/issues/31633))
* A new command was added for exporting Terraform function signatures in machine-readable format: `terraform metadata functions -json` ([#32487](https://github.com/hashicorp/terraform/issues/32487))
* The "Failed to install provider" error message now includes the reason a provider could not be installed. ([#31898](https://github.com/hashicorp/terraform/issues/31898))
* backend/gcs: Add `kms_encryption_key` argument, to allow encryption of state files using Cloud KMS keys. ([#24967](https://github.com/hashicorp/terraform/issues/24967))
* backend/gcs: Add `storage_custom_endpoint` argument, to allow communication with the backend via a Private Service Connect endpoint. ([#28856](https://github.com/hashicorp/terraform/issues/28856))
* backend/gcs: Update documentation for usage of `gcs` with `terraform_remote_state` ([#32065](https://github.com/hashicorp/terraform/issues/32065))
* backend/gcs: Update storage package to v1.28.0 ([#29656](https://github.com/hashicorp/terraform/issues/29656))
* When removing a workspace from the `cloud` backend `terraform workspace delete` will use Terraform Cloud's [Safe Delete](https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspaces#safe-delete-a-workspace) API if the `-force` flag is not provided. ([#31949](https://github.com/hashicorp/terraform/pull/31949))
* backend/oss: More robustly handle endpoint retrieval error ([#32295](https://github.com/hashicorp/terraform/issues/32295))
* local-exec provisioner: Added `quiet` argument. If `quiet` is set to `true`, Terraform will not print the entire command to stdout during plan. ([#32116](https://github.com/hashicorp/terraform/issues/32116))
* backend/http: Add support for mTLS authentication. ([#31699](https://github.com/hashicorp/terraform/issues/31699))
* cloud: Add support for using the [generic hostname](https://developer.hashicorp.com/terraform/cloud-docs/registry/using#generic-hostname-terraform-enterprise) localterraform.com in module and provider sources as a substitute for the currently configured cloud backend hostname. This enhancement was also applied to the remote backend.
* `terraform show` will now print an explanation when called on a Terraform workspace with empty state detailing why no resources are shown. ([#32629](https://github.com/hashicorp/terraform/issues/32629))
* backend/gcs: Added support for `GOOGLE_BACKEND_IMPERSONATE_SERVICE_ACCOUNT` env var to allow impersonating a different service account when `GOOGLE_IMPERSONATE_SERVICE_ACCOUNT` is configured for the GCP provider. ([#32557](https://github.com/hashicorp/terraform/issues/32557))
* backend/cos: Add support for the `assume_role` authentication method with the `tencentcloud` provider. This can be configured via the Terraform config or environment variables.
* backend/cos: Add support for the `security_token` authentication method with the `tencentcloud` provider. This can be configured via the Terraform config or environment variables.


EXPERIMENTS:

* Since its introduction the `yamlencode` function's documentation carried a warning that it was experimental. This predated our more formalized idea of language experiments and so wasn't guarded by an explicit opt-in, but the intention was to allow for small adjustments to its behavior if we learned it was producing invalid YAML in some cases, due to the relative complexity of the YAML specification.

    From Terraform v1.4 onwards, `yamlencode` is no longer documented as experimental and is now subject to the Terraform v1.x Compatibility Promises. There are no changes to its previous behavior in v1.3 and so no special action is required when upgrading.

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.3](https://github.com/hashicorp/terraform/blob/v1.3/CHANGELOG.md)
* [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
