## 1.4.0 (Unreleased)

UPGRADE NOTES:

- config: The `textencodebase64` function when called with encoding "GB18030" will now encode the euro symbol € as the two-byte sequence `0xA2,0xE3`, as required by the GB18030 standard, before applying base64 encoding.
- config: The `textencodebase64` function when called with encoding "GBK" or "CP936" will now encode the euro symbol € as the single byte `0x80` before applying base64 encoding. This matches the behavior of the Windows API when encoding to this Windows-specific character encoding.
- `terraform init`: When interpreting the hostname portion of a provider source address or the address of a module in a module registry, Terraform will now use _non-transitional_ IDNA2008 mapping rules instead of the transitional mapping rules previously used.

    This matches a change to [the WHATWG URL spec's rules for interpreting non-ASCII domain names](https://url.spec.whatwg.org/#concept-domain-to-ascii) which is being gradually adopted by web browsers. Terraform aims to follow the interpretation of hostnames used by web browsers for consistency. For some hostnames containing non-ASCII characters this may cause Terraform to now request a different "punycode" hostname when resolving.

BUG FIXES:

* The module installer will now record in its manifest a correct module source URL after normalization when the URL given as input contains both a query string portion and a subdirectory portion. Terraform itself doesn't currently make use of this information and so this is just a cosmetic fix to make the recorded metadata more correct. ([#31636](https://github.com/hashicorp/terraform/issues/31636))
* config: The `yamldecode` function now correctly handles entirely-nil YAML documents. Previously it would incorrectly return an unknown value instead of a null value. It will now return a null value as documented. ([#32151](https://github.com/hashicorp/terraform/issues/32151))
* Ensure correct ordering between data sources and the deletion of managed resource dependencies. ([#32209](https://github.com/hashicorp/terraform/issues/32209))
* Fix Terraform creating objects that should not exist in variables that specify default attributes in optional objects. ([#32178](https://github.com/hashicorp/terraform/issues/32178))
* Fix several Terraform crashes that are caused by HCL creating objects that should not exist in variables that specify default attributes in optional objects within collections. ([#32178](https://github.com/hashicorp/terraform/issues/32178))
* Fix inconsistent behaviour in empty vs null collections. ([#32178](https://github.com/hashicorp/terraform/issues/32178))

ENHANCEMENTS:

* `terraform_data` is a new builtin managed resource type, which can replace the use of `null_resource`, and can store data of any type [GH-31757]
* `terraform init` will now ignore entries in the optional global provider cache directory unless they match a checksum already tracked in the current configuration's dependency lock file. This therefore avoids the long-standing problem that when installing a new provider for the first time from the cache we can't determine the full set of checksums to include in the lock file. Once the lock file has been updated to include a checksum covering the item in the global cache, Terraform will then use the cache entry for subsequent installation of the same provider package. ([#32129](https://github.com/hashicorp/terraform/issues/32129))
* The "Failed to install provider" error message now includes the reason a provider could not be installed. ([#31898](https://github.com/hashicorp/terraform/issues/31898))
* backend/gcs: Add `kms_encryption_key` argument, to allow encryption of state files using Cloud KMS keys. ([#24967](https://github.com/hashicorp/terraform/issues/24967))
* backend/gcs: Add `storage_custom_endpoint` argument, to allow communication with the backend via a Private Service Connect endpoint. ([#28856](https://github.com/hashicorp/terraform/issues/28856))
* backend/gcs: Update documentation for usage of `gcs` with `terraform_remote_state` ([#32065](https://github.com/hashicorp/terraform/issues/32065))
* backed/gcs: Update storage package to v1.28.0 ([#29656](https://github.com/hashicorp/terraform/issues/29656))
* When removing a workspace from the `cloud` backend `terraform workspace delete` will use Terraform Cloud's [Safe Delete](https://developer.hashicorp.com/terraform/cloud-docs/api-docs/workspaces#safe-delete-a-workspace) API if the `-force` flag is not provided. ([#31949](https://github.com/hashicorp/terraform/pull/31949))
* backend/oss: More robustly handle endpoint retrieval error ([#32295](https://github.com/hashicorp/terraform/issues/32295))

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
