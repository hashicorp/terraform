## 1.11.0 (Unreleased)


NEW FEATURES:

* Add write-only attributes to resources. Providers can specify that certain attributes are write-only. They are not persisted in state. You can use ephemeral values in write-only attributes. ([#36031](https://github.com/hashicorp/terraform/issues/36031))

* `terraform test`: The `-junit-xml` option for the terraform test command is now generally available. This option allows the command to create a test report in JUnit XML format. Feedback during the experimental phase helped map terraform test concepts to the JUnit XML format, and new additons may happen in future releases. ([#36324](https://github.com/hashicorp/terraform/issues/36324))


ENHANCEMENTS:

* `init`: Provider installation will utilise credentials configured in a `.netrc` file for the download and shasum URLs returned by provider registries. ([#35843](https://github.com/hashicorp/terraform/issues/35843))

* New command `modules -json`: Displays a full list of all installed modules in a working directory, including whether each module is currently referenced by the working directory's configuration. ([#35884](https://github.com/hashicorp/terraform/issues/35884))

* `terraform test`: Test runs now support using mocked or overridden values during unit test runs (e.g., with command = "plan"). Set `override_during = plan` in the test configuration to use the overridden values during the plan phase. The default value is `override_during = apply`. ([#36227](https://github.com/hashicorp/terraform/issues/36227))

* `terraform test`: Add new `state_key` attribute for `run` blocks, allowing test authors control over which internal state file should be used for the current test run. ([#36185](https://github.com/hashicorp/terraform/issues/36185))


BUG FIXES:

* Updated dependency `github.com/hashicorp/go-slug` `v0.16.0` => `v0.16.3` to integrate latest changes (fix for CVE-2025-0377) ([#36273](https://github.com/hashicorp/terraform/issues/36273))


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
