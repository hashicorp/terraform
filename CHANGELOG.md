## 1.16.0 (Unreleased)


NEW FEATURES:

* Store PlannedPrivate data for providers ([#37986](https://github.com/hashicorp/terraform/issues/37986))

* New store block in terraform_data that can handle ephemeral and sensitive values ([#38298](https://github.com/hashicorp/terraform/issues/38298))

* Providers can now use nested blocks as computed values ([#38305](https://github.com/hashicorp/terraform/issues/38305))

* We now produce builds for Linux s390x (zLinux) ([#38384](https://github.com/hashicorp/terraform/issues/38384))

* workspace: The `workspace list` command can now produce machine-readable output when supplied with the `-json` flag ([#38397](https://github.com/hashicorp/terraform/issues/38397))


ENHANCEMENTS:

* feat(cli): terraform state show accepts a -json flag ([#23940](https://github.com/hashicorp/terraform/issues/23940))

* Show info when resources are left behind due to skip_cleanup ([#38449](https://github.com/hashicorp/terraform/issues/38449))


BUG FIXES:

* import blocks no longer ignore provider local names ([#38338](https://github.com/hashicorp/terraform/issues/38338))


UPGRADE NOTES:

* Provisioner bastion_host_key is now correctly applied. Existing usage of bastion_host_key should verify the configured key is correct. ([#38318](https://github.com/hashicorp/terraform/issues/38318))


EXPERIMENTS:

Experiments are only enabled in alpha releases of Terraform CLI. The following features are not yet available in stable releases.

- The experimental "deferred actions" feature, enabled by passing the `-allow-deferral` option to `terraform plan`, permits `count` and `for_each` arguments in `module`, `resource`, and `data` blocks to have unknown values and allows providers to react more flexibly to unknown values.
- `terraform test cleanup`: The experimental `test cleanup` command. In experimental builds of Terraform, a manifest file and state files for each failed cleanup operation during test operations are saved within the `.terraform` local directory. The `test cleanup` command will attempt to clean up the local state files left behind automatically, without requiring manual intervention.
- `terraform test`: `backend` blocks and `skip_cleanup` attributes:
  - Test authors can now specify `backend` blocks within `run` blocks in Terraform Test files. Run blocks with `backend` blocks will load state from the specified backend instead of starting from empty state on every execution. This allows test authors to keep long-running test infrastructure alive between test operations, saving time during regular test operations.
  - Test authors can now specify `skip_cleanup` attributes within test files and within run blocks. The `skip_cleanup` attribute tells `terraform test` not to clean up state files produced by run blocks with this attribute set to true. The state files for affected run blocks will be written to disk within the `.terraform` directory, where they can then be cleaned up manually using the also experimental `terraform test cleanup` command.

## Previous Releases

For information on prior major and minor releases, refer to their changelogs:

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
