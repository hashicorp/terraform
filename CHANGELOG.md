## 1.9.8 (October 16, 2024)

BUG FIXES:

* init: Highlight missing subdirectories of registry modules in error message ([#35848](https://github.com/hashicorp/terraform/pull/35848))
* init: Prevent crash when loading `provider_meta` blocks with invalid names ([#35842](https://github.com/hashicorp/terraform/pull/35842))
* config generation: Escape all invalid syntax in generate map keys with quotes ([#35837](https://github.com/hashicorp/terraform/pull/35837))
* plan: also validate provider requirements from state ([#35864](https://github.com/hashicorp/terraform/pull/35864))

## 1.9.7 (October 2, 2024)

BUG FIXES:

* config generation: escape map keys with whitespaces ([#35754](https://github.com/hashicorp/terraform/pull/35754))

## 1.9.6 (September 18, 2024)

BUG FIXES:

* plan renderer: Render complete changes within unknown nested blocks. ([#35644](https://github.com/hashicorp/terraform/issues/35644))
* plan renderer: Fix crash when attempting to render unknown nested blocks that contain attributes forcing resource replacement. ([#35644](https://github.com/hashicorp/terraform/issues/35644))
* plan renderer: Fix crash when rendering a plan that contains null attributes being update to unknown values. ([#35709](https://github.com/hashicorp/terraform/issues/35709))

## 1.9.5 (August 20, 2024)

ENHANCEMENTS:

* cloud: The cloud block can now interact with workspaces that have HCP resource IDs. ([#35495](https://github.com/hashicorp/terraform/issues/35495))

BUG FIXES:

* core: `removed` blocks with provisioners were not executed when the resource was in a nested module. ([#35611](https://github.com/hashicorp/terraform/issues/35611))

## 1.9.4 (August 7, 2024)

BUG FIXES:

* core: Unneeded variable validations were being executed during a destroy plan, which could cause plans starting with incomplete state to fail. ([#35511](https://github.com/hashicorp/terraform/issues/35511))
* init: Don't crash when discovering invalid syntax in duplicate required_providers blocks. ([#35533](https://github.com/hashicorp/terraform/issues/35533))

## 1.9.3 (July 24, 2024)

ENHANCEMENTS:

* Terraform now returns a more specific error message in the awkward situation where an input variable validation rule is known to have failed (`condition` returned `false`) but the error message is derived from an unknown value. ([#35400](https://github.com/hashicorp/terraform/pull/35400))

BUG FIXES:

* core: Terraform no longer performs an unnecessary refresh when removing an instance targeted by a `removed` block. ([#35458](https://github.com/hashicorp/terraform/pull/35458))
* config generation: Fix validation error when using nested computed or deprecated attributes. ([#35484](https://github.com/hashicorp/terraform/pull/35484))
* Updated to newer github.com/hashicorp/go-retryablehttp version, addressing CVE-2024-6104, and bringing in updates for several indirect dependencies. ([#35473](https://github.com/hashicorp/terraform/pull/35473))
* Moved to building with Go 1.22.5, which addresses CVE-2024-24791 and several other non-security bugs. ([#35494](https://github.com/hashicorp/terraform/pull/35494))

## 1.9.2 (July 10, 2024)

BUG FIXES:

* core: Fix panic when self-referencing direct instances from `count` and `for_each` meta attributes. ([#35432](https://github.com/hashicorp/terraform/pull/35432))

## 1.9.1 (July 3, 2024)

UPGRADE NOTES:

* Library used by Terraform (hashicorp/go-getter) for installing/updating modules was upgraded from v1.7.4 to v1.7.5. This addresses [CVE-2024-6257](https://nvd.nist.gov/vuln/detail/CVE-2024-6257). This change may have a negative effect on performance of `terraform init` or `terraform get` in case of larger git repositories. Please do file an issue if you find the performance difference noticable. ([#35376](https://github.com/hashicorp/terraform/pull/35376))

BUG FIXES:

* `terraform test`: Removed additional erroneous error message when referencing attributes that don't exist. ([#35408](https://github.com/hashicorp/terraform/pull/35408))
* `import` blocks: Fix crash that occurs when incorrectly referencing the `to` resource from the `id` attribute. ([#35420](https://github.com/hashicorp/terraform/pull/35420))

## 1.9.0 (June 26, 2024)

If you are upgrading from an earlier minor release, please refer to [the Terraform v1.9 Upgrade Guide](https://developer.hashicorp.com/terraform/language/v1.9.x/upgrade-guides).

NEW FEATURES:

* **Input variable validation rules can refer to other objects**: Previously input variable validation rules could refer only to the variable being validated. Now they are general expressions, similar to those elsewhere in a module, which can refer to other input variables and to other objects such as data resources.
* **`templatestring` function**: a new built-in function which is similar to `templatefile` but designed to render templates obtained dynamically, such as from a data resource result.

ENHANCEMENTS:

* `terraform plan`: Improved presentation of OPA and Sentinel policy evaluations in HCP Terraform remote runs, for logical separation.
* `terraform init` now accepts a `-json` option. If specified, enables the machine readable JSON output. ([#34886](https://github.com/hashicorp/terraform/pull/34886))
* `terraform test`: Test runs can now pass sensitive values to input variables while preserving their dynamic sensitivity. Previously sensitivity would be preserved only for variables statically declared as being sensitive, using `sensitive = true`. ([#35021](https://github.com/hashicorp/terraform/pull/35021))
* config: Input variable validation rules can now refer to other objects in the same module. ([#34955](https://github.com/hashicorp/terraform/pull/34955))
* config: `templatestring` function allows rendering a template provided as a string. ([#34968](https://github.com/hashicorp/terraform/pull/34968), [#35224](https://github.com/hashicorp/terraform/pull/35224), [#35285](https://github.com/hashicorp/terraform/pull/35285))
* core: Performance improvement during graph building for configurations with an extremely large number of `resource` blocks. ([#35088](https://github.com/hashicorp/terraform/pull/35088))
* built-in `terraform` provider: Allows `moved` block refactoring from the `hashicorp/null` provider `null_resource` resource type to the `terraform_data` resource type. ([#35163](https://github.com/hashicorp/terraform/pull/35163))
* `terraform output` with `cloud` block: Terraform no longer suggests that data loss could occur when outputs are not available. ([#35143](https://github.com/hashicorp/terraform/issues/35143))
* `terraform console`: Now has basic support for multi-line input in interactive mode. ([#34822](https://github.com/hashicorp/terraform/pull/34822))
    If an entered line contains opening parentheses/etc that are not closed, Terraform will await another line of input to complete the expression. This initial implementation is primarily intended to support pasting in multi-line expressions from elsewhere, rather than for manual multi-line editing, so the interactive editing support is currently limited.
* cli: Reduced copying of state to improve performance with large numbers of resources. ([#35164](https://github.com/hashicorp/terraform/issues/35164))
* `removed` blocks can now declare destroy-time provisioners which will be executed when the associated resource instances are destroyed. ([#35230](https://github.com/hashicorp/terraform/issues/35230))

BUG FIXES:

* `remote-exec` provisioner: Each remote connection will now be closed immediately after use. ([#34137](https://github.com/hashicorp/terraform/issues/34137))
* backend/s3: Fixed the digest value displayed for DynamoDB/S3 state checksum mismatches. ([#34387](https://github.com/hashicorp/terraform/issues/34387))
* `terraform test`: Fix bug in which non-Hashicorp providers required by testing modules and initialised within the test files were assigned incorrect registry addresses. ([#35161](https://github.com/hashicorp/terraform/issues/35161))
* config: The `templatefile` function no longer returns a "panic" error if the template file path is marked as sensitive. Instead, the template rendering result is also marked as sensitive. ([#35180](https://github.com/hashicorp/terraform/issues/35180))
* config: `import` blocks which referenced resources in non-existent modules were silently ignored when they should have raised an error ([#35330](https://github.com/hashicorp/terraform/issues/35330))
* `terraform init`: When selecting a version for a provider that has both positive and negative version constraints for the same prerelease -- e.g. `1.2.0-beta.1, !1.2.0-beta.1` -- the negative constraint will now overrule the positive, for consistency with how negative constraints are handled otherwise. Previously Terraform would incorrectly treat the positive as overriding the negative if the specified version was a prerelease. ([#35181](https://github.com/hashicorp/terraform/issues/35181))
* `import`: `import` blocks could block a destroy operation if the target resource was already deleted ([#35272](https://github.com/hashicorp/terraform/issues/35272))
* `cli`: plan output was missing blocks which were entirely unknown ([#35271](https://github.com/hashicorp/terraform/issues/35271))
* `cli`: fix crash when running `providers mirror` with an incomplete lock file ([#35322](https://github.com/hashicorp/terraform/issues/35322))
* core: Changing `create_before_destroy` when replacing an instance, then applying with `-refresh=false` would order the apply operations incorrectly ([#35261](https://github.com/hashicorp/terraform/issues/35261))
* core: Resource addresses that start with the optional `resource.` prefix will now be correctly parsed when used as an address target. ([#35333](https://github.com/hashicorp/terraform/issues/35333))

UPGRADE NOTES:

* `terraform test`: It is no longer valid to specify version constraints within provider blocks within .tftest.hcl files. Instead, version constraints must be supplied within the main configuration where the provider is in use.
* `import`: Invalid `import` blocks pointing to nonexistent modules were mistakenly ignored in prior versions. These will need to be fixed or removed in v1.9.

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.8](https://github.com/hashicorp/terraform/blob/v1.8/CHANGELOG.md)
* [v1.7](https://github.com/hashicorp/terraform/blob/v1.7/CHANGELOG.md)
* [v1.6](https://github.com/hashicorp/terraform/blob/v1.6/CHANGELOG.md)
* [v1.5](https://github.com/hashicorp/terraform/blob/v1.5/CHANGELOG.md)
* [v1.4](https://github.com/hashicorp/terraform/blob/v1.4/CHANGELOG.md)
* [v1.3](https://github.com/hashicorp/terraform/blob/v1.3/CHANGELOG.md)
* [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
