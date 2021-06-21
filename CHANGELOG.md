## 1.0.1 (Unreleased)

ENHANCEMENTS:

* json-output: The JSON plan output now indicates which state values are sensitive. [GH-28889]

BUG FIXES:

* backend/remote: Fix faulty Terraform Cloud version check when migrating state to the remote backend with multiple local workspaces [GH-28864]
* cli: Fix crash with deposed instances in json plan output [GH-28922]
* core: Fix crash when provider modifies and unknown block during plan [GH-28941]
* core: Diagnostic context was missing for some errors when validating blocks [GH-28979]
* core: Fix crash when calling `setproduct` with unknown values [GH-28984]
* json-output: Fix an issue where the JSON configuration representation was missing fully-unwrapped references. [GH-8884]
* json-output: Fix JSON plan resource drift to remove unchanged resources. [GH-28975]

## 1.0.0 (June 08, 2021)

Terraform v1.0 is an unusual release in that its primary focus is on stability, and it represents the culmination of several years of work in previous major releases to make sure that the Terraform language and internal architecture will be a suitable foundation for forthcoming additions that will remain backward compatible.

Terraform v1.0.0 intentionally has no significant changes compared to Terraform v0.15.5. You can consider the v1.0 series as a direct continuation of the v0.15 series; we do not intend to issue any further releases in the v0.15 series, because all of the v1.0 releases will be only minor updates to address bugs.

For all future minor releases with major version 1, we intend to preserve backward compatibility as described in detail in [the Terraform v1.0 Compatibility Promises](https://www.terraform.io/docs/language/v1-compatibility-promises.html). The later Terraform v1.1.0 will, therefore, be the first minor release with new features that we will implement with consideration of those promises.

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
