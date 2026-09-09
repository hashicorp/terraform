## 1.17.0-beta1 (September 09, 2026)


NEW FEATURES:

* Terraform now supports variables and locals in provider requirements ([#39153](https://github.com/hashicorp/terraform/issues/39153))

* A new `-minimal-refresh` planning option has been added, which will only refresh resources that have proposed changes. ([#35290](https://github.com/hashicorp/terraform/issues/35290))

* policy: Terraform Policy is now generally available. The `-policies` flag for `plan`, `apply`, and `init` no longer requires the `-allow-experimental-features` flag. See https://developer.hashicorp.com/terraform/policy for more information. ([#38970](https://github.com/hashicorp/terraform/issues/38970))


ENHANCEMENTS:

* command/init: Enrich log messages with provider versions ([#38918](https://github.com/hashicorp/terraform/issues/38918))

* test: Add mock_provider support for ephemeral resources ([#38928](https://github.com/hashicorp/terraform/issues/38928))

* command/login: display warning after successful login if user is subject to an organization's TTL policy


BUG FIXES:

* funcs: pow and log no longer panic when result is not a number ([#38912](https://github.com/hashicorp/terraform/issues/38912))

* ephemeral: Terraform will now use and display diagnostics raised when _renewing_ an ephemeral resource. This may cause warnings to appear that previously were lost. We expect that any error diagnostics that were previously lost would have caused confusing downstream errors, so we do not anticipate this change to be breaking. ([#38989](https://github.com/hashicorp/terraform/issues/38989))

* test: Deterministic dependency ordering in cleanup graph ([#38247](https://github.com/hashicorp/terraform/issues/38247))

* query: report Unknown and N/A results for policy evaluations of discovered resources ([#39058](https://github.com/hashicorp/terraform/issues/39058))


NOTES:

* version: JSON output now includes a new `format_version` field, which will enable safer future changes of the command's JSON output format. It is assumed existing tooling ignores unknown fields and therefore this change should not be breaking in itself but we advice consumers to pay attention to `format_version` in future releases and/or use latest version of hashicorp/terraform-json & hashicorp/terraform-exec which does. ([#38930](https://github.com/hashicorp/terraform/issues/38930))


## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

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
