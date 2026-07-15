## 1.16.0 (Unreleased)


NEW FEATURES:

* Store PlannedPrivate data for providers ([#37986](https://github.com/hashicorp/terraform/issues/37986))

* New store block in terraform_data that can handle ephemeral and sensitive values ([#38298](https://github.com/hashicorp/terraform/issues/38298))

* Providers can now use nested blocks as computed values ([#38305](https://github.com/hashicorp/terraform/issues/38305))

* import: add support for import blocks inside modules ([#38352](https://github.com/hashicorp/terraform/issues/38352))

* We now produce builds for Linux s390x (zLinux) ([#38384](https://github.com/hashicorp/terraform/issues/38384))

* workspace: The `workspace list` command can now produce machine-readable output when supplied with the `-json` flag ([#38397](https://github.com/hashicorp/terraform/issues/38397))

* Resource action triggers can now use `on_failure` modes of `halt`, `taint`, or `continue` ([#38722](https://github.com/hashicorp/terraform/issues/38722))


ENHANCEMENTS:

* feat(cli): terraform state show accepts a -json flag ([#23940](https://github.com/hashicorp/terraform/issues/23940))

* Show info when resources are left behind due to skip_cleanup ([#38449](https://github.com/hashicorp/terraform/issues/38449))

* Action configuration now has a new `caller` symbol which contains the object value from the calling resource. ([#38668](https://github.com/hashicorp/terraform/issues/38668))

* Actions can now use before_destroy and after_destroy events ([#38668](https://github.com/hashicorp/terraform/issues/38668))

* cloud: Render a summary of Terraform policy evaluation outcomes for plan and apply runs against HCP Terraform ([#38715](https://github.com/hashicorp/terraform/issues/38715))

* policy: Resolve the policy plugin entitlement (host, token, organization) from the configured cloud/remote backend for init, plan, and apply, instead of the plugin reading credentials itself ([#38716](https://github.com/hashicorp/terraform/issues/38716))

* The 'terraform graph' command now accepts a -format flag, and can output graphs in Mermaid format ([#38719](https://github.com/hashicorp/terraform/issues/38719))

* child module outputs with unreferenced deprecated nested attributes no longer return deprecation warnings. ([#38778](https://github.com/hashicorp/terraform/issues/38778))

* Support destroy=false in resource lifecycle blocks. ([#38784](https://github.com/hashicorp/terraform/issues/38784))

* contains() function can now test for null ([#38792](https://github.com/hashicorp/terraform/issues/38792))

* The `terraform console` command now accepts an optional `-scope=<module address>` flag, which can be used to evaluate expressions within the scope of a module or a specific module instance. ([#31861](https://github.com/hashicorp/terraform/issues/31861))

* If `-invoke` results in multiple resource calls triggering the action, it can now be combined with `-target` to specify the calling resource instance ([#38845](https://github.com/hashicorp/terraform/issues/38845))


BUG FIXES:

* import blocks no longer ignore provider local names ([#38338](https://github.com/hashicorp/terraform/issues/38338))

* Fix a `terraform apply` panic when the plan contained a no-op change for a deposed object on a resource whose configuration declared a `lifecycle.precondition` or `lifecycle.postcondition` ([#38586](https://github.com/hashicorp/terraform/issues/38586))

* workspace: Terraform will now error if an invalid workspace name becomes selected due to actions performed out-of-band ([#38594](https://github.com/hashicorp/terraform/issues/38594))

* test: Terraform will now raise a warning when a file referenced via `-filter` flag does not exist. ([#38603](https://github.com/hashicorp/terraform/issues/38603))

* init: Stop removing locks from the dependency lock file corresponding to providers configured as a dev_override ([#38634](https://github.com/hashicorp/terraform/issues/38634))

* init: Add warnings when unmanaged providers are in use and will impact provider installation processes. ([#38656](https://github.com/hashicorp/terraform/issues/38656))

* Actions are now invoked with respect to all resource dependencies. ([#38668](https://github.com/hashicorp/terraform/issues/38668))

* return correct error when import target exists in state, but not config ([#38782](https://github.com/hashicorp/terraform/issues/38782))

* merge no longer panics with null objects ([#38792](https://github.com/hashicorp/terraform/issues/38792))


NOTES:

* init: Errors due to incompatible `-upgrade` and `-lockfile=readonly` flags are now raised earlier in the init process. ([#38561](https://github.com/hashicorp/terraform/issues/38561))


UPGRADE NOTES:

* Provisioner bastion_host_key is now correctly applied. Existing usage of bastion_host_key should verify the configured key is correct. ([#38318](https://github.com/hashicorp/terraform/issues/38318))


EXPERIMENTS:

Experiments are only enabled in alpha releases of Terraform CLI. The following features are not yet available in stable releases.

- The experimental "deferred actions" feature, enabled by passing the `-allow-deferral` option to `terraform plan`, permits `count` and `for_each` arguments in `module`, `resource`, and `data` blocks to have unknown values and allows providers to react more flexibly to unknown values.
- `terraform test cleanup`: The experimental `test cleanup` command. In experimental builds of Terraform, a manifest file and state files for each failed cleanup operation during test operations are saved within the `.terraform` local directory. The `test cleanup` command will attempt to clean up the local state files left behind automatically, without requiring manual intervention.
- `terraform test`: `backend` blocks and `skip_cleanup` attributes:
  - Test authors can now specify `backend` blocks within `run` blocks in Terraform Test files. Run blocks with `backend` blocks will load state from the specified backend instead of starting from empty state on every execution. This allows test authors to keep long-running test infrastructure alive between test operations, saving time during regular test operations.
  - Test authors can now specify `skip_cleanup` attributes within test files and within run blocks. The `skip_cleanup` attribute tells `terraform test` not to clean up state files produced by run blocks with this attribute set to true. The state files for affected run blocks will be written to disk within the `.terraform` directory, where they can then be cleaned up manually using the also experimental `terraform test cleanup` command.
- `terraform query`: The experimental `-policies` flag permits specifying one or more policy set directory paths to evaluate policies against resources discovered by list blocks during a query operation.

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
