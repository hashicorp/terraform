## 1.0.1 (Unreleased)

BUG FIXES:

* backend/remote: Fix faulty Terraform Cloud version check when migrating state to the remote backend with multiple local workspaces [GH-28864]

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
