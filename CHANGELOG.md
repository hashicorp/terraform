## 1.13.0 (Unreleased)


NEW FEATURES:

* The new command `terraform stacks` exposes some stack operations through the cli. The available subcommands depend on the stacks plugin implementation. Use `terraform stacks -help` to see available commands. ([#36931](https://github.com/hashicorp/terraform/issues/36931))

* Deferred actions: The `plan`, `apply`, and `refresh` commands now support the `-allow-deferral` flag. The flag enables Terraform and Terraform Providers to defer changes with unresolvable unknown values to future plans instead of failing the entire plan. ([#37067](https://github.com/hashicorp/terraform/issues/37067))


ENHANCEMENTS:

* Filesystem functions are now checked for consistent results to catch invalid data during apply ([#37001](https://github.com/hashicorp/terraform/issues/37001))

* Allow successful init when provider constraint matches at least one valid version ([#37137](https://github.com/hashicorp/terraform/issues/37137))


NOTES:

* The command `terraform rpcapi` is now generally available. It is not intended for public consumption, but exposes certain Terraform operations through an RPC interface compatible with [go-plugin](https://github.com/hashicorp/go-plugin). ([#37067](https://github.com/hashicorp/terraform/issues/37067))



## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

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
