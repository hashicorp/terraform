## 1.16.0-beta1 (July 23, 2026)


NEW FEATURES:

* Terraform now stores planned private data for providers, allowing provider-specific state to be preserved across plan and apply. ([#37986](https://github.com/hashicorp/terraform/issues/37986))

* `terraform_data`: The new `store` block can hold ephemeral and sensitive values across plan and apply. ([#38298](https://github.com/hashicorp/terraform/issues/38298))

* Providers can now use nested blocks as computed values ([#38305](https://github.com/hashicorp/terraform/issues/38305))

* import: `import` blocks inside modules are now supported. ([#38352](https://github.com/hashicorp/terraform/issues/38352))

* Terraform is now available as a pre-built binary for Linux s390x (zLinux). ([#38384](https://github.com/hashicorp/terraform/issues/38384))

* Resource action triggers can now use `on_failure` modes of `halt`, `taint`, or `continue`. ([#38722](https://github.com/hashicorp/terraform/issues/38722))


ENHANCEMENTS:

* state show: The `state show` command can now produce machine-readable output when supplied with the `-json` flag ([#23940](https://github.com/hashicorp/terraform/issues/23940))

* workspace: The `workspace list` command can now produce machine-readable output when supplied with the `-json` flag ([#38397](https://github.com/hashicorp/terraform/issues/38397))

* test: Terraform now reports which resources were left behind when `skip_cleanup` is set. ([#38449](https://github.com/hashicorp/terraform/issues/38449))

* stacks: Action configurations now have access to a `caller` symbol containing the object value of the calling resource. ([#38668](https://github.com/hashicorp/terraform/issues/38668))

* Actions can now use `before_destroy` and `after_destroy` events. ([#38668](https://github.com/hashicorp/terraform/issues/38668))

* cloud: Terraform now displays a summary of policy evaluation outcomes for `plan` and `apply` runs against HCP Terraform. ([#38715](https://github.com/hashicorp/terraform/issues/38715))

* policy: Terraform now resolves policy plugin credentials from the configured cloud or remote backend during `init`, `plan`, and `apply`, rather than requiring the plugin to read credentials itself. ([#38716](https://github.com/hashicorp/terraform/issues/38716))

* graph: The `terraform graph` command can now output graphs in Mermaid format using the `-format=mermaid` flag. ([#38719](https://github.com/hashicorp/terraform/issues/38719))

* Child module outputs with unreferenced deprecated nested attributes no longer return deprecation warnings. ([#38778](https://github.com/hashicorp/terraform/issues/38778))

* Resource `lifecycle` blocks now support `destroy = false` to prevent a resource from being destroyed. ([#38784](https://github.com/hashicorp/terraform/issues/38784))

* The `contains()` function can now test for `null` values. ([#38792](https://github.com/hashicorp/terraform/issues/38792))

* console: The `terraform console` command now accepts an optional `-scope=<module address>` flag, which can be used to evaluate expressions within the scope of a module or a specific module instance. ([#31861](https://github.com/hashicorp/terraform/issues/31861))

* `-invoke` can now be combined with `-target` to specify the calling resource instance when multiple resources trigger the same action. ([#38845](https://github.com/hashicorp/terraform/issues/38845))

* The `terraform stacks` command now automatically infers the target hostname from the local credentials file (`credentials.tfrc.json`) when neither `TF_STACKS_HOSTNAME` nor `TF_CLOUD_HOSTNAME` is set ([#38896](https://github.com/hashicorp/terraform/issues/38896))


BUG FIXES:

* `import` blocks now correctly respect provider local names. ([#38338](https://github.com/hashicorp/terraform/issues/38338))

* `terraform apply` no longer panics when the plan contains a no-op change for a deposed resource that has `lifecycle.precondition` or `lifecycle.postcondition` blocks. ([#38586](https://github.com/hashicorp/terraform/issues/38586))

* workspace: Terraform now raises an error if an invalid workspace name becomes selected due to out-of-band changes. ([#38594](https://github.com/hashicorp/terraform/issues/38594))

* test: Terraform now raises a warning when a file referenced via the `-filter` flag does not exist. ([#38603](https://github.com/hashicorp/terraform/issues/38603))

* init: Terraform no longer removes locks from the dependency lock file for providers configured as `dev_override`. ([#38634](https://github.com/hashicorp/terraform/issues/38634))

* init: Terraform now warns when unmanaged providers are in use and may impact provider installation. ([#38656](https://github.com/hashicorp/terraform/issues/38656))

* Actions are now invoked with respect to all resource dependencies. ([#38668](https://github.com/hashicorp/terraform/issues/38668))

* Terraform now returns the correct error when an `import` target exists in state but has no corresponding configuration. ([#38782](https://github.com/hashicorp/terraform/issues/38782))

* The `merge()` function no longer panics when passed `null` objects. ([#38792](https://github.com/hashicorp/terraform/issues/38792))


NOTES:

* init: Errors due to incompatible `-upgrade` and `-lockfile=readonly` flags are now raised earlier in the init process. ([#38561](https://github.com/hashicorp/terraform/issues/38561))


UPGRADE NOTES:

* `bastion_host_key` is now correctly applied by provisioners. Review your provisioner configurations to verify the configured key is correct before upgrading. ([#38318](https://github.com/hashicorp/terraform/issues/38318))


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
