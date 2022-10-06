## 1.4.0 (Unreleased)

BUG FIXES:

* The module installer will now record in its manifest a correct module source URL after normalization when the URL given as input contains both a query string portion and a subdirectory portion. Terraform itself doesn't currently make use of this information and so this is just a cosmetic fix to make the recorded metadata more correct. [GH-31636]

ENHANCEMENTS:

* The "Failed to install provider" error message now includes the reason a provider could not be installed. [GH-31898]
* backend/gcs: Add `kms_encryption_key` argument, to allow encryption of state files using Cloud KMS keys. [GH-24967]

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
