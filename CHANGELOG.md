## 1.11.1 (Unreleased)

## 1.11.0 (February 27, 2025)


NEW FEATURES:

* Add write-only attributes to resources. Providers can specify that certain attributes are write-only. They are not persisted in state. You can use ephemeral values in write-only attributes. ([#36031](https://github.com/hashicorp/terraform/issues/36031))

* `terraform test`: The `-junit-xml` option for the terraform test command is now generally available. This option allows the command to create a test report in JUnit XML format. Feedback during the experimental phase helped map terraform test concepts to the JUnit XML format, and new additons may happen in future releases. ([#36324](https://github.com/hashicorp/terraform/issues/36324))

* S3 native state locking is now generally available. The `use_lockfile` argument enables users to adopt the S3-native mechanism for state locking. As part of this change, we've deprecated the DynamoDB-related arguments in favor of this new locking mechanism. While you can still use DynamoDB alongside S3-native state locking for migration purposes, we encourage migrating to the new state locking mechanism. ([#36338](https://github.com/hashicorp/terraform/issues/36338))


ENHANCEMENTS:

* `init`: Provider installation will utilise credentials configured in a `.netrc` file for the download and shasum URLs returned by provider registries. ([#35843](https://github.com/hashicorp/terraform/issues/35843))

* `terraform test`: Test runs now support using mocked or overridden values during unit test runs (e.g., with command = "plan"). Set `override_during = plan` in the test configuration to use the overridden values during the plan phase. The default value is `override_during = apply`. ([#36227](https://github.com/hashicorp/terraform/issues/36227))

* `terraform test`: Add new `state_key` attribute for `run` blocks, allowing test authors control over which internal state file should be used for the current test run. ([#36185](https://github.com/hashicorp/terraform/issues/36185))

* Updates the azure backend authentication to match the terraform-provider-azurermprovider authentication, in several ways:
    - github.com/hashicorp/go-azure-helpers: v0.43.0 -> v0.71.0
    - github.com/hashicorp/go-azure-sdk/[resource-manager/sdk]: v0.20241212.1154051. This replaces the deprecated Azure SDK used before
    - github.com/jackofallops/giovanni: v0.15.1 -> v0.27.0. Meanwhile, updating the azure storage API version from 2018-11-09 to 2023-11-03
    - Following new properties are added for the azure backend configuration:
        - use_cli
        - use_aks_workload_identity
        - client_id_file_path
        - client_certificate
        - client_id_file_path
        - client_secret_file_path
 ([#36258](https://github.com/hashicorp/terraform/issues/36258))

* Include `ca-certificates` package in our official Docker image to help with certificate handling by downstream ([#36486](https://github.com/hashicorp/terraform/issues/36486))


BUG FIXES:

* ephemeral values: correct error message when ephemeral values are included in provisioner output ([#36427](https://github.com/hashicorp/terraform/issues/36427))

* Attempting to override a variable during `apply` via `TF_VAR_` environment variable will now yield warning instead of misleading error. ([#36435](https://github.com/hashicorp/terraform/issues/36435))

* backends: Fix crash when interrupting during interactive prompt for values ([#36448](https://github.com/hashicorp/terraform/issues/36448))

* Fixes hanging behavior seen when applying a saved plan with -auto-approve using the cloud backend ([#36453](https://github.com/hashicorp/terraform/issues/36453))
## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

- [v1.10](https://github.com/hashicorp/terraform/blob/v1.10/CHANGELOG.md)
- [v1.9](https://github.com/hashicorp/terraform/blob/v1.9/CHANGELOG.md)
- [v1.8](https://github.com/hashicorp/terraform/blob/v1.8/CHANGELOG.md)
- [v1.7](https://github.com/hashicorp/terraform/blob/v1.7/CHANGELOG.md)
- [v1.6](https://github.com/hashicorp/terraform/blob/v1.6/CHANGELOG.md)
- [v1.5](https://github.com/hashicorp/terraform/blob/v1.5/CHANGELOG.md)
- [v1.4](https://github.com/hashicorp/terraform/blob/v1.4/CHANGELOG.md)
- [v1.3](https://github.com/hashicorp/terraform/blob/v1.3/CHANGELOG.md)
- [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
- [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
- [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
- [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
- [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
- [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
- [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
- [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
