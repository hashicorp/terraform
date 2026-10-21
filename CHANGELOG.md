## 1.18.0 (October 21, 2026)


ENHANCEMENTS:

* genconfig: use hcl-native printing instead of falling back to go string ([#39193](https://github.com/hashicorp/terraform/issues/39193))

* terraform init: Requests to a provider registry now reuse a single connection instead of opening a new one for every request ([#39262](https://github.com/hashicorp/terraform/issues/39262))


BUG FIXES:

* `textdecodebase64`: Accept valid UTF-8 strings containing the U+FFFD replacement character ([#39256](https://github.com/hashicorp/terraform/issues/39256))

* fix(cli): properly validate apply time -targets match plan -targets ([#39257](https://github.com/hashicorp/terraform/issues/39257))

* fix(fileset): skip dangling symlinks and other matched paths that no longer resolve, while continuing to return other regular-file matches. ([#39260](https://github.com/hashicorp/terraform/issues/39260))

* version: The `version` command now returns diagnostics in JSON format when the `-json` flag is supplied. ([#39394](https://github.com/hashicorp/terraform/issues/39394))


NOTES:

* init: Terraform will no longer produce a duplicated version log message in JSON output when configuring a backend for the first time. ([#39266](https://github.com/hashicorp/terraform/issues/39266))

* modules: The `modules` command now shows diagnostics in its JSON format output ([#39396](https://github.com/hashicorp/terraform/issues/39396))

* test: The -help output now lists the -run-parallelism option ([#39384](https://github.com/hashicorp/terraform/issues/39384))


## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

- [v1.17](https://github.com/hashicorp/terraform/blob/v1.17/CHANGELOG.md)
- [v1.16](https://github.com/hashicorp/terraform/blob/v1.16/CHANGELOG.md)
- [v1.15](https://github.com/hashicorp/terraform/blob/v1.15/CHANGELOG.md)
- [v1.14](https://github.com/hashicorp/terraform/blob/v1.14/CHANGELOG.md)
- [v1.13](https://github.com/hashicorp/terraform/blob/v1.13/CHANGELOG.md)
- [v1.12](https://github.com/hashicorp/terraform/blob/v1.12/CHANGELOG.md)
- [v1.11](https://github.com/hashicorp/terraform/blob/v1.11/CHANGELOG.md)
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
