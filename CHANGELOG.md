## 0.10.8 (Unreleased)

NEW FEATURES:

* **New `etcdv3` backend**, for use with the newer etcd api [GH-15680]
* **New interpolation function `chunklist`**, for spliting a list into a list of lists with specified sublist chunk sizes. [GH-15112]

IMPROVEMENTS:

* backend/s3: Add options to skip AWS validation which allows non-AWS S3 backends [GH-15553]

BUG FIXES:

* provisioner/chef: Clean clients from chef-vault when recreate_client enabled [GH-16357]
## 0.10.7 (October 2, 2017)

NEW FEATURES:

* Provider plugins can now optionally be cached in a shared directory to avoid re-downloading them for each configuration working directory. For more information, see [the documentation](https://github.com/hashicorp/terraform/blob/34956cd12449cb77db3f55e3286cd369e8332eeb/website/docs/configuration/providers.html.md#provider-plugin-cache). ([#16000](https://github.com/hashicorp/terraform/issues/16000))

IMPROVEMENTS:

* config: New `abs` interpolation function, returning the absolute value of a number ([#16168](https://github.com/hashicorp/terraform/issues/16168))
* config: New `transpose` interpolation function, which swaps the keys and values in a map of lists of strings. ([#16192](https://github.com/hashicorp/terraform/issues/16192))
* cli: The Terraform CLI now supports tab-completion for commands and certain arguments for `bash` and `zsh` users. See [the tab-completion docs](https://github.com/hashicorp/terraform/blob/2c782e60fad78e6fc976d850162322608f074e57/website/docs/commands/index.html.markdown#shell-tab-completion) for information on how to enable it. ([#16176](https://github.com/hashicorp/terraform/issues/16176))
* cli: `terraform state rm` now includes in its output the count of resources that were removed from the state. ([#16137](https://github.com/hashicorp/terraform/issues/16137))

BUG FIXES:

* modules: Update go-getter to fix crash when fetching invalid source subdir ([#16161](https://github.com/hashicorp/terraform/issues/16161))
* modules: Fix regression in the handling of modules sourcing other modules with relative paths ([#16160](https://github.com/hashicorp/terraform/issues/16160))
* core: Skip local value interpolation during destroy ([#16213](https://github.com/hashicorp/terraform/issues/16213))

## 0.10.6 (September 19, 2017)

UPGRADE NOTES:

* The internal storage of modules has changed in this release, so after
  upgrading `terraform init` must be run to re-install modules in the new
  on-disk format. The existing installed versions of modules will be ignored,
  so the latest version of each module will be installed.

IMPROVEMENTS:

* Modules can now be installed from [the Terraform Registry](https://registry.terraform.io/)
* cli: `terraform import` now accepts an option `-allow-missing-config` that overrides the default requirement that a configuration block must already be present for the resource being imported. ([#15876](https://github.com/hashicorp/terraform/issues/15876))

## 0.10.5 (September 14, 2017)

NEW FEATURES:

* config: `indent` interpolation function appends spaces to all but the first line of a multi-line string ([#15311](https://github.com/hashicorp/terraform/issues/15311))

IMPROVEMENTS:

* cli: `terraform fmt` has a new option `-check` which makes it return a non-zero exit status if any formatting changes are required ([#15387](https://github.com/hashicorp/terraform/issues/15387))
* cli: When [running Terraform in automation](https://www.terraform.io/guides/running-terraform-in-automation.html), a new environment variable `TF_IN_AUTOMATION` can be used to disable or adjust certain prompts that would normally include specific CLI commands to run. This assumes that the wrapping automation tool is providing its own UI for guiding the user through the workflow, and thus the standard advice would be redundant and/or confusing. ([#16059](https://github.com/hashicorp/terraform/issues/16059))

BUG FIXES:

* cli: restore the "(forces new resource)" annotations on attributes that were inadvertently disabled in 0.10.4. ([#16067](https://github.com/hashicorp/terraform/issues/16067))
* cli: fix regression with installing modules from git when the `GIT_SSH_COMMAND` environment variable is set ([#16099](https://github.com/hashicorp/terraform/issues/16099))

## 0.10.4 (September 6, 2017)

IMPROVEMENTS:
* `terraform apply` now uses the standard resource address syntax to refer to resources in its log ([#15884](https://github.com/hashicorp/terraform/issues/15884))
* `terraform plan` output has some minor adjustments to improve readability and accessibility for those who can't see its colors ([#15884](https://github.com/hashicorp/terraform/issues/15884))

BUG FIXES:

* backend/consul: fix crash during consul backend initialization ([#15976](https://github.com/hashicorp/terraform/issues/15976))
* backend/azurerm: ensure that blob storage metadata is preserved when updating state blobs, to avoid losing track of lock metadata ([#16015](https://github.com/hashicorp/terraform/issues/16015))
* config: local values now work properly in resource `count` and in modules with more than one `.tf` file ([#15995](https://github.com/hashicorp/terraform/issues/15995)] [[#15982](https://github.com/hashicorp/terraform/issues/15982))
* cli: removed some inconsistencies in how data sources are counted and tallied in plan vs. apply and apply vs. destroy. In particular, data sources are no longer incorrectly counted as destroyed in `terraform destroy` ([#15884](https://github.com/hashicorp/terraform/issues/15884))

## 0.10.3 (August 30, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

* LGPL Dependencies Removed ([#15862](https://github.com/hashicorp/terraform/issues/15862))

NEW FEATURES:

* **Local Values**: this new configuration language feature allows assigning a symbolic local name to an expression so it can be used multiple times in configuration without repetition. See [the documentation](https://github.com/hashicorp/terraform/blob/master/website/docs/configuration/locals.html.md) for how to define and use local values. ([#15449](https://github.com/hashicorp/terraform/issues/15449))
* **`base64gzip` interpolation function**: compresses a string with gzip and then base64-encodes the result ([#3858](https://github.com/hashicorp/terraform/issues/3858))
* **`flatten` interpolation function**: turns a list of lists, or list of lists of lists, etc into a flat list of primitive values ([#15278](https://github.com/hashicorp/terraform/issues/15278))
* **`urlencode` interpolation function**: applies standard URL encoding to a string so that it can be embedded in a URL without making it invalid and without any of the characters being interpreted as part of the URL structure ([#15871](https://github.com/hashicorp/terraform/issues/15871))
* **`salt-masterless` provisioner**: runs Salt in masterless mode on a target server ([#14720](https://github.com/hashicorp/terraform/issues/14720))

IMPROVEMENTS:

* config: The `jsonencode` interpolation function now accepts nested list and map structures, where before it would accept only strings, lists of strings, and maps of strings. ([#14884](https://github.com/hashicorp/terraform/issues/14884))
* cli: The "creation complete" (and similar) messages from `terraform apply` now include a total elapsed time for each operation. ([#15548](https://github.com/hashicorp/terraform/issues/15548))
* cli: Module installation (with either `terraform init` or `terraform get`) now detects and recursively initializes submodules when the source is a git repository. ([#15891](https://github.com/hashicorp/terraform/issues/15891))
* cli: Modules can now be installed from `.tar.xz` archives, in addition to the existing `.tar.gz`, `.tar.bz2` and `.zip`. ([#15891](https://github.com/hashicorp/terraform/issues/15891))
* provisioner/local-exec: now possible to specify a custom "interpreter", overriding the default of either `bash -c` (on Unix) or `cmd.exe /C` (on Windows) ([#15166](https://github.com/hashicorp/terraform/issues/15166))
* backend/consul: can now set the path to a specific CA certificate file, client certificate file, and client key file that will be used when configuring the underlying Consul client. ([#15405](https://github.com/hashicorp/terraform/issues/15405))
* backend/http: now has optional support for locking, with special support from the target server. Additionally, the update operation can now optionally be implemented via `PUT` rather than `POST`. ([#15793](https://github.com/hashicorp/terraform/issues/15793))
* helper/resource: Add `TestStep.SkipFunc` ([#15957](https://github.com/hashicorp/terraform/issues/15957))

BUG FIXES:

* cli: `terraform init` now verifies the required Terraform version from the root module config. Previously this was verified only on subsequent commands, after initialization. ([#15935](https://github.com/hashicorp/terraform/issues/15935))
* cli: `terraform validate` now consults `terraform.tfvars`, if present, to set variable values. This is now consistent with the behavior of other commands. ([#15938](https://github.com/hashicorp/terraform/issues/15938))

## 0.10.2 (August 16, 2017)

BUG FIXES:

* tools/terraform-bundle: Add missing Ui to ProviderInstaller (fix crash) ([#15826](https://github.com/hashicorp/terraform/issues/15826))
* go-plugin: don't crash when server emits non-key-value JSON ([go-plugin#43](https://github.com/hashicorp/go-plugin/pull/43))

## 0.10.1 (August 15, 2017)

BUG FIXES:

* Fix `terraform state rm` and `mv` commands to work correctly with remote state backends ([#15652](https://github.com/hashicorp/terraform/issues/15652))
* Fix errors when interpolations fail during input ([#15780](https://github.com/hashicorp/terraform/issues/15780))
* Backoff retries in remote-execution provisioner ([#15772](https://github.com/hashicorp/terraform/issues/15772))
* Load plugins from `~/.terraform.d/plugins/OS_ARCH/` and `.terraformrc` ([#15769](https://github.com/hashicorp/terraform/issues/15769))
* The `import` command was ignoring the remote state configuration ([#15768](https://github.com/hashicorp/terraform/issues/15768))
* Don't allow leading slashes in s3 bucket names for remote state ([#15738](https://github.com/hashicorp/terraform/issues/15738))

IMPROVEMENTS:

* helper/schema: Add `GetOkExists` schema function ([#15723](https://github.com/hashicorp/terraform/issues/15723))
* helper/schema: Make 'id' a reserved field name ([#15695](https://github.com/hashicorp/terraform/issues/15695))
* command/init: Display version + source when initializing plugins ([#15804](https://github.com/hashicorp/terraform/issues/15804))

INTERNAL CHANGES:

* DiffFieldReader.ReadField caches results to optimize deeply nested schemas ([#15663](https://github.com/hashicorp/terraform/issues/15663))


## 0.10.0 (August 2, 2017)

**This is the complete 0.9.11 to 0.10.0 CHANGELOG**

BACKWARDS INCOMPATIBILITIES / NOTES:

* A new flag `-auto-approve` has been added to `terraform apply`. This flag controls whether an interactive approval is applied before
  making the changes in the plan. For now this flag defaults to `true` to preserve previous behavior, but this will become the new default
  in a future version. We suggest that anyone running `terraform apply` in wrapper scripts or automation refer to the upgrade guide to learn
  how to prepare such wrapper scripts for the later breaking change.
* The `validate` command now checks that all variables are specified by default.  The validation will fail by default if that's not the
  case. ([#13872](https://github.com/hashicorp/terraform/issues/13872))
* `terraform state rm` now requires at least one argument. Previously, calling it with no arguments would remove all resources from state,
  which is consistent with the other `terraform state` commands but unlikely enough that we considered it better to be inconsistent here to
  reduce the risk of accidentally destroying the state.
* Terraform providers are no longer distributed as part of the main Terraform distribution. Instead, they are installed automatically as
  part of running `terraform init`. It is therefore now mandatory to run `terraform init` before any other operations that use provider
  plugins, to ensure that the required plugins are installed and properly initialized.
* The `terraform env` family of commands have been renamed to `terraform workspace`, in response to feedback that the previous naming was
  confusing due to collisions with other concepts of the same name. The commands still work the same as they did before, and the `env`
  subcommand is still supported as an alias for backward compatibility. The `env` subcommand will be removed altogether in a future release,
  so it's recommended to update any automation or wrapper scripts that use these commands.
* The `terraform init` subcommand no longer takes a SOURCE argument to copy to the current directory. The behavior has been changed to match
  that of `plan` and `apply`, so that a configuration can be provided as an argument on the commandline while initializing the current
  directory. If a module needs to be copied into the current directory before initialization, it will have to be done manually.
* The `-target` option available on several Terraform subcommands has changed behavior and **now matches potentially more resources**.  In
  particular, given an option `-target=module.foo`, resources in any descendent modules of `foo` will also be targeted, where before this
  was not true. After upgrading, be sure to look carefully at the set of changes proposed by `terraform plan` when using `-target` to ensure
  that the target is being interpreted as expected. Note that the `-target` argument is offered for exceptional circumstances only and is
  not intended for routine use.
* The `import` command requires that imported resources be specified in the configuration file. Previously, users were encouraged to import
  a resource and _then_ write the configuration block for it. This creates the risk that users could import a resource and subsequently
  create no configuration for it, which results in Terraform deleting the resource. If the imported resource is not present in the
  configuration file, the `import` command will fail.

FEATURES:

* **Separate Provider Releases:** Providers are now released independently from Terraform.
* **Automatic Provider Installation:** The required providers will be automatically installed during `terraform init`.
* **Provider Constraints:** Provider are now versioned, and version constraints may be declared in the configuration.

PROVIDERS:

* Providers now maintain their own CHANGELOGs in their respective repositories: [terraform-providers](https://github.com/terraform-providers)

IMPROVEMENTS:

* cli: Add a `-from-module` flag to `terraform init` to re-introduce the legacy `terraform init` behavior of fetching a module. ([#15666](https://github.com/hashicorp/terraform/issues/15666))
* backend/s3: Add `workspace_key_prefix` to allow a user-configurable prefix for workspaces in the S3 Backend. ([#15370](https://github.com/hashicorp/terraform/issues/15370))
* cli: `terraform apply` now has an option `-auto-approve=false` that produces an interactive prompt to approve the generated plan. This will become the default workflow in a future Terraform version. ([#7251](https://github.com/hashicorp/terraform/issues/7251))
* cli: `terraform workspace show` command prints the current workspace name in a way that's more convenient for processing in wrapper scripts. ([#15157](https://github.com/hashicorp/terraform/issues/15157))
* cli: `terraform state rm` will now generate an error if no arguments are passed, whereas before it treated it as an open resource address selecting _all_ resources ([#15283](https://github.com/hashicorp/terraform/issues/15283))
* cli: Files in the config directory ending in `.auto.tfvars` are now loaded automatically (in lexicographical order) in addition to the single `terraform.tfvars` file that auto-loaded previously. ([#13306](https://github.com/hashicorp/terraform/issues/13306))
* Providers no longer in the main Terraform distribution; installed automatically by init instead ([#15208](https://github.com/hashicorp/terraform/issues/15208))
* cli: `terraform env` command renamed to `terraform workspace` ([#14952](https://github.com/hashicorp/terraform/issues/14952))
* cli: `terraform init` command now has `-upgrade` option to download the latest versions (within specified constraints) of modules and provider plugins.
* cli: The `-target` option to various Terraform operation can now target resources in descendent modules. ([#15314](https://github.com/hashicorp/terraform/issues/15314))
* cli: Minor updates to `terraform plan` output: use standard resource address syntax, more visually-distinct `-/+` actions, and more ([#15362](https://github.com/hashicorp/terraform/issues/15362))
* config: New interpolation function `contains`, to find if a given string exists in a list of strings. ([#15322](https://github.com/hashicorp/terraform/issues/15322))

BUG FIXES:

* provisioner/chef: fix panic ([#15617](https://github.com/hashicorp/terraform/issues/15617))
* Don't show a message about the path to the state file if the state is remote ([#15435](https://github.com/hashicorp/terraform/issues/15435))
* Fix crash when `terraform graph` is run with no configuration ([#15588](https://github.com/hashicorp/terraform/issues/15588))
* Handle correctly the `.exe` suffix on locally-compiled provider plugins on Windows systems. ([#15587](https://github.com/hashicorp/terraform/issues/15587))
* config: Fixed a parsing issue in the interpolation language HIL that was causing misinterpretation of literal strings ending with escaped backslashes ([#15415](https://github.com/hashicorp/terraform/issues/15415))
* core: the S3 Backend was failing to remove the state file checksums from DynamoDB when deleting a workspace ([#15383](https://github.com/hashicorp/terraform/issues/15383))
* core: Improved reslience against crashes for a certain kind of inconsistency in the representation of list values in state. ([#15390](https://github.com/hashicorp/terraform/issues/15390))
* core: Display correct to and from backends in copy message when migrating to new remote state ([#15318](https://github.com/hashicorp/terraform/issues/15318))
* core: Fix a regression from 0.9.6 that was causing the tally of resources to create to be double-counted sometimes in the plan output ([#15344](https://github.com/hashicorp/terraform/issues/15344))
* cli: the state `rm` and `mv` commands were always loading a state from a Backend, and ignoring the `-state` flag ([#15388](https://github.com/hashicorp/terraform/issues/15388))
* cli: certain prompts in `terraform init` were respecting `-input=false` but not the `TF_INPUT` environment variable ([#15391](https://github.com/hashicorp/terraform/issues/15391))
* state: Further work, building on [#15423](https://github.com/hashicorp/terraform/issues/15423), to improve the internal design of the state managers to make this code more maintainable and reduce the risk of regressions; this may lead to slight changes to the number of times Terraform writes to remote state and how the serial is implemented with respect to those writes, which does not affect outward functionality but is worth noting if you log or inspect state updates for debugging purposes.
* config: Interpolation function `cidrhost` was not correctly calcluating host addresses under IPv6 CIDR prefixes ([#15321](https://github.com/hashicorp/terraform/issues/15321))
* provisioner/chef: Prevent a panic while trying to read the connection info ([#15271](https://github.com/hashicorp/terraform/issues/15271))
* provisioner/file: Refactor the provisioner validation function to prevent false positives ([#15273](https://github.com/hashicorp/terraform/issues/15273))

INTERNAL CHANGES:

* helper/schema: Actively disallow reserved field names in schema ([#15522](https://github.com/hashicorp/terraform/issues/15522))
* helper/schema: Force field names to be alphanum lowercase + underscores ([#15562](https://github.com/hashicorp/terraform/issues/15562))


## 0.10.0-rc1 to 0.10.0 (August 2, 2017)

BUG FIXES:

* provisioner/chef: fix panic ([#15617](https://github.com/hashicorp/terraform/issues/15617))

IMPROVEMENTS:

* cli: Add a `-from-module` flag to `terraform init` to re-introduce the legacy `terraform init` behavior of fetching a module. ([#15666](https://github.com/hashicorp/terraform/issues/15666))


## 0.10.0-rc1 (July 19, 2017)

BUG FIXES:

* Don't show a message about the path to the state file if the state is remote ([#15435](https://github.com/hashicorp/terraform/issues/15435))
* Fix crash when `terraform graph` is run with no configuration ([#15588](https://github.com/hashicorp/terraform/issues/15588))
* Handle correctly the `.exe` suffix on locally-compiled provider plugins on Windows systems. ([#15587](https://github.com/hashicorp/terraform/issues/15587))

INTERNAL CHANGES:

* helper/schema: Actively disallow reserved field names in schema ([#15522](https://github.com/hashicorp/terraform/issues/15522))
* helper/schema: Force field names to be alphanum lowercase + underscores ([#15562](https://github.com/hashicorp/terraform/issues/15562))

## 0.10.0-beta2 (July 6, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

* A new flag `-auto-approve` has been added to `terraform apply`. This flag controls whether an interactive approval is applied before making the changes in the plan. For now this flag defaults to `true` to preserve previous behavior, but this will become the new default in a future version. We suggest that anyone running `terraform apply` in wrapper scripts or automation refer to the upgrade guide to learn how to prepare such wrapper scripts for the later breaking change.
* The `validate` command now checks that all variables are specified by default.
  The validation will fail by default if that's not the case. ([#13872](https://github.com/hashicorp/terraform/issues/13872))
* `terraform state rm` now requires at least one argument. Previously, calling it with no arguments would remove all resources from state, which is consistent with the other `terraform state` commands but unlikely enough that we considered it better to be inconsistent here to reduce the risk of accidentally destroying the state.

IMPROVEMENTS:

* backend/s3: Add `workspace_key_prefix` to allow a user-configurable prefix for workspaces in the S3 Backend. ([#15370](https://github.com/hashicorp/terraform/issues/15370))
* cli: `terraform apply` now has an option `-auto-approve=false` that produces an interactive prompt to approve the generated plan. This will become the default workflow in a future Terraform version. ([#7251](https://github.com/hashicorp/terraform/issues/7251))
* cli: `terraform workspace show` command prints the current workspace name in a way that's more convenient for processing in wrapper scripts. ([#15157](https://github.com/hashicorp/terraform/issues/15157))
* cli: `terraform state rm` will now generate an error if no arguments are passed, whereas before it treated it as an open resource address selecting _all_ resources ([#15283](https://github.com/hashicorp/terraform/issues/15283))
* cli: Files in the config directory ending in `.auto.tfvars` are now loaded automatically (in lexicographical order) in addition to the single `terraform.tfvars` file that auto-loaded previously. ([#13306](https://github.com/hashicorp/terraform/issues/13306))

BUG FIXES:

* config: Fixed a parsing issue in the interpolation language HIL that was causing misinterpretation of literal strings ending with escaped backslashes ([#15415](https://github.com/hashicorp/terraform/issues/15415))
* core: the S3 Backend was failing to remove the state file checksums from DynamoDB when deleting a workspace ([#15383](https://github.com/hashicorp/terraform/issues/15383))
* core: Improved reslience against crashes for a certain kind of inconsistency in the representation of list values in state. ([#15390](https://github.com/hashicorp/terraform/issues/15390))
* core: Display correct to and from backends in copy message when migrating to new remote state ([#15318](https://github.com/hashicorp/terraform/issues/15318))
* core: Fix a regression from 0.9.6 that was causing the tally of resources to create to be double-counted sometimes in the plan output ([#15344](https://github.com/hashicorp/terraform/issues/15344))
* cli: the state `rm` and `mv` commands were always loading a state from a Backend, and ignoring the `-state` flag ([#15388](https://github.com/hashicorp/terraform/issues/15388))
* cli: certain prompts in `terraform init` were respecting `-input=false` but not the `TF_INPUT` environment variable ([#15391](https://github.com/hashicorp/terraform/issues/15391))
* state: Further work, building on [#15423](https://github.com/hashicorp/terraform/issues/15423), to improve the internal design of the state managers to make this code more maintainable and reduce the risk of regressions; this may lead to slight changes to the number of times Terraform writes to remote state and how the serial is implemented with respect to those writes, which does not affect outward functionality but is worth noting if you log or inspect state updates for debugging purposes.

## 0.10.0-beta1 (June 22, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

* Terraform providers are no longer distributed as part of the main Terraform distribution. Instead, they are installed automatically
  as part of running `terraform init`. It is therefore now mandatory to run `terraform init` before any other operations that use
  provider plugins, to ensure that the required plugins are installed and properly initialized.
* The `terraform env` family of commands have been renamed to `terraform workspace`, in response to feedback that the previous naming
  was confusing due to collisions with other concepts of the same name. The commands still work the same as they did before, and
  the `env` subcommand is still supported as an alias for backward compatibility. The `env` subcommand will be removed altogether in
  a future release, so it's recommended to update any automation or wrapper scripts that use these commands.
* The `terraform init` subcommand no longer takes a SOURCE argument to copy to the current directory. The behavior has
  been changed to match that of `plan` and `apply`, so that a configuration can be provided as an argument on the
  commandline while initializing the current directory. If a module needs to be copied into the current directory before
  initialization, it will have to be done manually.
* The `-target` option available on several Terraform subcommands has changed behavior and **now matches potentially more resources**.
  In particular, given an option `-target=module.foo`, resources in any descendent modules of `foo` will also be targeted, where before
  this was not true. After upgrading, be sure to look carefully at the set of changes proposed by `terraform plan` when using `-target`
  to ensure that the target is being interpreted as expected. Note that the `-target` argument is offered for exceptional circumstances
  only and is not intended for routine use.
* The `import` command requires that imported resources be specified in the configuration file. Previously, users were encouraged to
  import a resource and _then_ write the configuration block for it. This creates the risk that users could import a resource and
  subsequently create no configuration for it, which results in Terraform deleting the resource. If the imported resource is not
  present in the configuration file, the `import` command will fail.

IMPROVEMENTS:

* Providers no longer in the main Terraform distribution; installed automatically by init instead ([#15208](https://github.com/hashicorp/terraform/issues/15208))
* cli: `terraform env` command renamed to `terraform workspace` ([#14952](https://github.com/hashicorp/terraform/issues/14952))
* cli: `terraform init` command now has `-upgrade` option to download the latest versions (within specified constraints) of modules and provider plugins.
* cli: The `-target` option to various Terraform operation can now target resources in descendent modules. ([#15314](https://github.com/hashicorp/terraform/issues/15314))
* cli: Minor updates to `terraform plan` output: use standard resource address syntax, more visually-distinct `-/+` actions, and more ([#15362](https://github.com/hashicorp/terraform/issues/15362))
* config: New interpolation function `contains`, to find if a given string exists in a list of strings. ([#15322](https://github.com/hashicorp/terraform/issues/15322))

BUG FIXES:

* config: Interpolation function `cidrhost` was not correctly calcluating host addresses under IPv6 CIDR prefixes ([#15321](https://github.com/hashicorp/terraform/issues/15321))
* provisioner/chef: Prevent a panic while trying to read the connection info ([#15271](https://github.com/hashicorp/terraform/issues/15271))
* provisioner/file: Refactor the provisioner validation function to prevent false positives ([#15273](https://github.com/hashicorp/terraform/issues/15273))

## 0.9.11 (Jul 3, 2017)

BUG FIXES:

* core: Hotfix for issue where a state from a plan was reported as not equal to the same state stored to a backend. This arose from the fix for the previous issue where the incorrect copy of the state was being used when applying with a plan. ([#15460](https://github.com/hashicorp/terraform/issues/15460))


## 0.9.10 (June 29, 2017)

BUG FIXES:

* core: Hotfix for issue where state index wasn't getting properly incremented when applying a change containing only data source updates and/or resource drift. (That is, state changes made during refresh.)
  This issue is significant only for the "atlas" backend, since that backend verifies on the server that state serial numbers are being used consistently. ([#15423](https://github.com/hashicorp/terraform/issues/15423))

## 0.9.9 (June 26, 2017)

BUG FIXES:

 * provisioner/file: Refactor the provisioner validation function to prevent false positives ([#15273](https://github.com/hashicorp/terraform/issues/15273)))
 * provisioner/chef: Prevent a panic while trying to read the connection info ([#15271](https://github.com/hashicorp/terraform/issues/15271)))

## 0.9.8 (June 7, 2017)

NOTE:

* The 0.9.7 release had a bug with its new feature of periodically persisting state to the backend during an apply, as part of [[#14834](https://github.com/hashicorp/terraform/issues/14834)]. This change has been reverted in this release and will be re-introduced at a later time once it has been made to work properly.

IMPROVEMENTS:

* provider/google: `network` argument in `google_compute_instance_group` is now optional ([#13493](https://github.com/hashicorp/terraform/issues/13493))
* provider/google: Add support for `draining_timeout_sec` to `google_compute_backend_service`. ([#14559](https://github.com/hashicorp/terraform/issues/14559))

BUG FIXES:

* provider/aws: fixed reading network configurations for `spot_fleet_request` ([#13748](https://github.com/hashicorp/terraform/issues/13748))

## 0.9.7 (June 7, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

* The `lock_table` attribute in the S3 backend configuration has been deprecated in favor of `dynamodb_table`, which better reflects that the table is no longer only used for locks. ([#14949](https://github.com/hashicorp/terraform/issues/14949))

FEATURES:

 * **New Data Source:** `aws_elastic_beanstalk_solution_stack` ([#14944](https://github.com/hashicorp/terraform/issues/14944))
 * **New Data Source:** `aws_elasticache_cluster` ([#14895](https://github.com/hashicorp/terraform/issues/14895))
 * **New Data Source:** `aws_ssm_parameter` ([#15035](https://github.com/hashicorp/terraform/issues/15035))
 * **New Data Source:** `azurerm_public_ip` ([#15110](https://github.com/hashicorp/terraform/issues/15110))
 * **New Resource:** `aws_ssm_parameter` ([#15035](https://github.com/hashicorp/terraform/issues/15035))
 * **New Resource:** `aws_ssm_patch_baseline` ([#14954](https://github.com/hashicorp/terraform/issues/14954))
 * **New Resource:** `aws_ssm_patch_group` ([#14954](https://github.com/hashicorp/terraform/issues/14954))
 * **New Resource:** `librato_metric` ([#14562](https://github.com/hashicorp/terraform/issues/14562))
 * **New Resource:** `digitalocean_certificate` ([#14578](https://github.com/hashicorp/terraform/issues/14578))
 * **New Resource:** `vcd_edgegateway_vpn` ([#13123](https://github.com/hashicorp/terraform/issues/13123))
 * **New Resource:** `vault_mount` ([#14456](https://github.com/hashicorp/terraform/issues/14456))
 * **New Interpolation Function:** `bcrypt` ([#14725](https://github.com/hashicorp/terraform/issues/14725))

IMPROVEMENTS:

* backend/consul: Storing state to Consul now uses Check-And-Set (CAS) by default to avoid inconsistent state, and will automatically attempt to re-acquire a lock if it is lost during Terraform execution. ([#14930](https://github.com/hashicorp/terraform/issues/14930))
* core: Remote state is now persisted more frequently to minimize data loss in the event of a crash. ([#14834](https://github.com/hashicorp/terraform/issues/14834))
* provider/alicloud: Add the function of replacing ecs instance's system disk ([#15048](https://github.com/hashicorp/terraform/issues/15048))
* provider/aws: Expose RDS instance and cluster resource id ([#14882](https://github.com/hashicorp/terraform/issues/14882))
* provider/aws: Export internal tunnel addresses + document ([#14835](https://github.com/hashicorp/terraform/issues/14835))
* provider/aws: Fix misleading error in aws_route validation ([#14972](https://github.com/hashicorp/terraform/issues/14972))
* provider/aws: Support import of aws_lambda_event_source_mapping ([#14898](https://github.com/hashicorp/terraform/issues/14898))
* provider/aws: Add support for a configurable timeout in db_option_group ([#15023](https://github.com/hashicorp/terraform/issues/15023))
* provider/aws: Add task_parameters parameter to aws_ssm_maintenance_window_task resource ([#15104](https://github.com/hashicorp/terraform/issues/15104))
* provider/aws: Expose reason of EMR cluster termination ([#15117](https://github.com/hashicorp/terraform/issues/15117))
* provider/aws: `data.aws_acm_certificate` can now filter by `type` ([#15063](https://github.com/hashicorp/terraform/issues/15063))
* provider/azurerm: Ignore case sensivity in Azurerm resource enums ([#14861](https://github.com/hashicorp/terraform/issues/14861))
* provider/digitalocean: Add support for changing TTL on DigitalOcean domain records. ([#14805](https://github.com/hashicorp/terraform/issues/14805))
* provider/google: Add ability to import Google Compute persistent disks ([#14573](https://github.com/hashicorp/terraform/issues/14573))
* provider/google: `google_container_cluster.master_auth` should be optional ([#14630](https://github.com/hashicorp/terraform/issues/14630))
* provider/google: Add CORS support for `google_storage_bucket` ([#14695](https://github.com/hashicorp/terraform/issues/14695))
* provider/google: Allow resizing of Google Cloud persistent disks ([#15077](https://github.com/hashicorp/terraform/issues/15077))
* provider/google: Add private_ip_google_access update support to google_compute_subnetwork ([#15125](https://github.com/hashicorp/terraform/issues/15125))
* provider/heroku: can now import Heroku Spaces ([#14973](https://github.com/hashicorp/terraform/issues/14973))
* provider/kubernetes: Upgrade K8S from 1.5.3 to 1.6.1 ([#14923](https://github.com/hashicorp/terraform/issues/14923))
* provider/kubernetes: Provide more details about why PVC failed to bind ([#15019](https://github.com/hashicorp/terraform/issues/15019))
* provider/kubernetes: Allow sourcing config_path from `KUBECONFIG` env var ([#14889](https://github.com/hashicorp/terraform/issues/14889))
* provider/openstack: Add support provider networks ([#10265](https://github.com/hashicorp/terraform/issues/10265))
* provider/openstack: Allow numerical protocols in security group rules ([#14917](https://github.com/hashicorp/terraform/issues/14917))
* provider/openstack: Sort request/response headers in debug output ([#14956](https://github.com/hashicorp/terraform/issues/14956))
* provider/openstack: Add support for FWaaS routerinsertion extension ([#12589](https://github.com/hashicorp/terraform/issues/12589))
* provider/openstack: Add Terraform version to UserAgent string ([#14955](https://github.com/hashicorp/terraform/issues/14955))
* provider/openstack: Optimize the printing of debug output ([#15086](https://github.com/hashicorp/terraform/issues/15086))
* provisioner/chef: Use `helpers.shema.Provisoner` in Chef provisioner V2 ([#14681](https://github.com/hashicorp/terraform/issues/14681))

BUG FIXES:

* provider/alicloud: set `alicloud_nat_gateway` zone to be Computed to avoid perpetual diffs ([#15050](https://github.com/hashicorp/terraform/issues/15050))
* provider/alicloud: set provider to read env vars for access key and secrey key if empty strings ([#15050](https://github.com/hashicorp/terraform/issues/15050))
* provider/alicloud: Fix vpc and vswitch bugs while creating vpc and vswitch ([#15082](https://github.com/hashicorp/terraform/issues/15082))
* provider/alicloud: Fix allocating public ip bug ([#15049](https://github.com/hashicorp/terraform/issues/15049))
* provider/alicloud: Fix security group rules nic_type bug ([#15114](https://github.com/hashicorp/terraform/issues/15114))
* provider/aws: ForceNew aws_launch_config on ebs_block_device change ([#14899](https://github.com/hashicorp/terraform/issues/14899))
* provider/aws: Avoid crash when EgressOnly IGW disappears ([#14929](https://github.com/hashicorp/terraform/issues/14929))
* provider/aws: Allow IPv6/IPv4 addresses to coexist ([#13702](https://github.com/hashicorp/terraform/issues/13702))
* provider/aws: Expect exception on deletion of APIG Usage Plan Key ([#14958](https://github.com/hashicorp/terraform/issues/14958))
* provider/aws: Fix panic on nil dead_letter_config ([#14964](https://github.com/hashicorp/terraform/issues/14964))
* provider/aws: Work around IAM eventual consistency in CW Log Subs ([#14959](https://github.com/hashicorp/terraform/issues/14959))
* provider/aws: Fix ModifyInstanceAttribute on new instances ([#14992](https://github.com/hashicorp/terraform/issues/14992))
* provider/aws: Fix issue with removing tags in aws_cloudwatch_log_group ([#14886](https://github.com/hashicorp/terraform/issues/14886))
* provider/aws: Raise timeout for VPC DHCP options creation to 5 mins ([#15084](https://github.com/hashicorp/terraform/issues/15084))
* provider/aws: Retry Redshift cluster deletion on InvalidClusterState ([#15068](https://github.com/hashicorp/terraform/issues/15068))
* provider/aws: Retry Lambda func creation on IAM error ([#15067](https://github.com/hashicorp/terraform/issues/15067))
* provider/aws: Retry ECS service creation on ClusterNotFoundException ([#15066](https://github.com/hashicorp/terraform/issues/15066))
* provider/aws: Retry ECS service update on ServiceNotFoundException ([#15073](https://github.com/hashicorp/terraform/issues/15073))
* provider/aws: Retry DB parameter group delete on InvalidDBParameterGroupState ([#15071](https://github.com/hashicorp/terraform/issues/15071))
* provider/aws: Guard against panic when no aws_default_vpc found ([#15070](https://github.com/hashicorp/terraform/issues/15070))
* provider/aws: Guard against panic if no NodeGroupMembers returned in `elasticache_replication_group` ([#13488](https://github.com/hashicorp/terraform/issues/13488))
* provider/aws: Revoke default ipv6 egress rule for aws_security_group ([#15075](https://github.com/hashicorp/terraform/issues/15075))
* provider/aws: Lambda ENI deletion fails on destroy ([#11849](https://github.com/hashicorp/terraform/issues/11849))
* provider/aws: Add gov and cn hosted zone Ids to aws_elb_hosted_zone data source ([#15149](https://github.com/hashicorp/terraform/issues/15149))
* provider/azurerm: VM - making `os_profile` optional ([#14176](https://github.com/hashicorp/terraform/issues/14176))
* provider/azurerm: Preserve the Subnet properties on Update ([#13877](https://github.com/hashicorp/terraform/issues/13877))
* provider/datadog: make datadog_user verified a computed attribute ([#15034](https://github.com/hashicorp/terraform/issues/15034))
* provider/datadog: use correct evaluation_delay parameter ([#14878](https://github.com/hashicorp/terraform/issues/14878))
* provider/digitalocean: Refresh DO loadbalancer from state if 404 ([#14897](https://github.com/hashicorp/terraform/issues/14897))
* provider/github: Do not set incorrect values in github_team data source ([#14859](https://github.com/hashicorp/terraform/issues/14859))
* provider/google: use a mutex to prevent concurrent sql instance operations ([#14424](https://github.com/hashicorp/terraform/issues/14424))
* provider/google: Set instances to computed in compute_instance_group ([#15025](https://github.com/hashicorp/terraform/issues/15025))
* provider/google: Make google_compute_autoscaler use Update instead of Patch. ([#15101](https://github.com/hashicorp/terraform/issues/15101))
* provider/kubernetes: Ignore internal k8s labels in `kubernetes_persistent_volume` ([#13716](https://github.com/hashicorp/terraform/issues/13716))
* provider/librato: Add retry to librato_alert ([#15118](https://github.com/hashicorp/terraform/issues/15118))
* provider/postgresql: Fix for leaking credentials in the provider ([#14817](https://github.com/hashicorp/terraform/issues/14817))
* provider/postgresql: Drop the optional WITH token from CREATE ROLE. ([#14864](https://github.com/hashicorp/terraform/issues/14864))
* provider/rancher: refresh rancher_host from state on nil or removed host ([#15015](https://github.com/hashicorp/terraform/issues/15015))

## 0.9.6 (May 25, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

* When assigning a "splat variable" to a resource attribute, like `foo = "${some_resource.foo.*.baz}"`, it is no longer required (nor recommended) to wrap the string in list brackets. The extra brackets continue to be allowed for resource attributes for compatibility, but this will cease to be allowed in a future version. ([#14737](https://github.com/hashicorp/terraform/issues/14737))
* provider/aws: Allow lightsail resources to work in other regions. Previously Terraform would automatically configure lightsail resources to run solely in `us-east-1`. This means that if a provider was initialized with a different region than `us-east-1`, users will need to create a provider alias to maintain their lightsail resources in us-east-1 [[#14685](https://github.com/hashicorp/terraform/issues/14685)].
* provider/aws: Users of `aws_cloudfront_distribution` `default_cache_behavior` will notice that cookies is now a required value - even if that value is none ([#12628](https://github.com/hashicorp/terraform/issues/12628))
* provider/google: Users of `google_compute_health_check` who were not setting a value for the `host` property of `http_health_check` or `https_health_check` previously had a faulty default value. This has been fixed and will show as a change in terraform plan/apply. ([#14441](https://github.com/hashicorp/terraform/issues/14441))

FEATURES:

* **New Provider:** `ovh` ([#12669](https://github.com/hashicorp/terraform/issues/12669))
* **New Resource:** `aws_default_subnet` ([#14476](https://github.com/hashicorp/terraform/issues/14476))
* **New Resource:** `aws_default_vpc` ([#11710](https://github.com/hashicorp/terraform/issues/11710))
* **New Resource:** `aws_default_vpc_dhcp_options` ([#14475](https://github.com/hashicorp/terraform/issues/14475))
* **New Resource:** `aws_devicefarm_project` ([#14288](https://github.com/hashicorp/terraform/issues/14288))
* **New Resource:** `aws_wafregional_ipset` ([#13705](https://github.com/hashicorp/terraform/issues/13705))
* **New Resource:** `aws_wafregional_byte_match_set` ([#13705](https://github.com/hashicorp/terraform/issues/13705))
* **New Resource:** `azurerm_express_route_circuit` ([#14265](https://github.com/hashicorp/terraform/issues/14265))
* **New Resource:** `gitlab_deploy_key` ([#14734](https://github.com/hashicorp/terraform/issues/14734))
* **New Resource:** `gitlab_group` ([#14490](https://github.com/hashicorp/terraform/issues/14490))
* **New Resource:** `google_compute_router` ([#12411](https://github.com/hashicorp/terraform/issues/12411))
* **New Resource:** `google_compute_router_interface` ([#12411](https://github.com/hashicorp/terraform/issues/12411))
* **New Resource:** `google_compute_router_peer` ([#12411](https://github.com/hashicorp/terraform/issues/12411))
* **New Resource:** `kubernetes_horizontal_pod_autoscaler` ([#14763](https://github.com/hashicorp/terraform/issues/14763))
* **New Resource:** `kubernetes_service` ([#14554](https://github.com/hashicorp/terraform/issues/14554))
* **New Resource:** `openstack_dns_zone_v2` ([#14721](https://github.com/hashicorp/terraform/issues/14721))
* **New Resource:** `openstack_dns_recordset_v2` ([#14813](https://github.com/hashicorp/terraform/issues/14813))
* **New Data Source:** `aws_db_snapshot` ([#10291](https://github.com/hashicorp/terraform/issues/10291))
* **New Data Source:** `aws_kms_ciphertext` ([#14691](https://github.com/hashicorp/terraform/issues/14691))
* **New Data Source:** `github_user` ([#14570](https://github.com/hashicorp/terraform/issues/14570))
* **New Data Source:** `github_team` ([#14614](https://github.com/hashicorp/terraform/issues/14614))
* **New Data Source:** `google_storage_object_signed_url` ([#14643](https://github.com/hashicorp/terraform/issues/14643))
* **New Interpolation Function:** `pow` ([#14598](https://github.com/hashicorp/terraform/issues/14598))

IMPROVEMENTS:

* core: After `apply`, if the state cannot be persisted to remote for some reason then write out a local state file for recovery ([#14423](https://github.com/hashicorp/terraform/issues/14423))
* core: It's no longer required to surround an attribute value that is just a "splat" variable with a redundant set of array brackets. ([#14737](https://github.com/hashicorp/terraform/issues/14737))
* core/provider-split: Split out the Oracle OPC provider to new structure ([#14362](https://github.com/hashicorp/terraform/issues/14362))
* provider/aws: Show state reason when EC2 instance fails to launch ([#14479](https://github.com/hashicorp/terraform/issues/14479))
* provider/aws: Show last scaling activity when ASG creation/update fails ([#14480](https://github.com/hashicorp/terraform/issues/14480))
* provider/aws: Add `tags` (list of maps) for `aws_autoscaling_group` ([#13574](https://github.com/hashicorp/terraform/issues/13574))
* provider/aws: Support filtering in ASG data source ([#14501](https://github.com/hashicorp/terraform/issues/14501))
* provider/aws: Add ability to 'terraform import' aws_kms_alias resources ([#14679](https://github.com/hashicorp/terraform/issues/14679))
* provider/aws: Allow lightsail resources to work in other regions ([#14685](https://github.com/hashicorp/terraform/issues/14685))
* provider/aws: Configurable timeouts for EC2 instance + spot instance ([#14711](https://github.com/hashicorp/terraform/issues/14711))
* provider/aws: Add ability to define timeouts for DMS replication instance ([#14729](https://github.com/hashicorp/terraform/issues/14729))
* provider/aws: Add support for X-Ray tracing to aws_lambda_function ([#14728](https://github.com/hashicorp/terraform/issues/14728))
* provider/azurerm: Virtual Machine Scale Sets with managed disk support ([#13717](https://github.com/hashicorp/terraform/issues/13717))
* provider/azurerm: Virtual Machine Scale Sets with single placement option support ([#14510](https://github.com/hashicorp/terraform/issues/14510))
* provider/azurerm: Adding support for VMSS Data Disks using Managed Disk feature ([#14608](https://github.com/hashicorp/terraform/issues/14608))
* provider/azurerm: Adding support for 4TB disks ([#14688](https://github.com/hashicorp/terraform/issues/14688))
* provider/cloudstack: Load the provider configuration from a CloudMonkey config file ([#13926](https://github.com/hashicorp/terraform/issues/13926))
* provider/datadog: Add last aggregator to datadog_timeboard resource ([#14391](https://github.com/hashicorp/terraform/issues/14391))
* provider/datadog: Added new evaluation_delay parameter ([#14433](https://github.com/hashicorp/terraform/issues/14433))
* provider/docker: Allow Windows Docker containers to map volumes ([#13584](https://github.com/hashicorp/terraform/issues/13584))
* provider/docker: Add `network_alias` to `docker_container` resource ([#14710](https://github.com/hashicorp/terraform/issues/14710))
* provider/fastly: Mark the `s3_access_key`, `s3_secret_key`, & `secret_key` fields as sensitive ([#14634](https://github.com/hashicorp/terraform/issues/14634))
* provider/gitlab: Add namespcace ID attribute to `gitlab_project` ([#14483](https://github.com/hashicorp/terraform/issues/14483))
* provider/google: Add a `url` attribute to `google_storage_bucket` ([#14393](https://github.com/hashicorp/terraform/issues/14393))
* provider/google: Make google resource storage bucket importable ([#14455](https://github.com/hashicorp/terraform/issues/14455))
* provider/google: Add support for privateIpGoogleAccess on subnetworks ([#14234](https://github.com/hashicorp/terraform/issues/14234))
* provider/google: Add import support to `google_sql_user` ([#14457](https://github.com/hashicorp/terraform/issues/14457))
* provider/google: add failover parameter to `google_sql_database_instance` ([#14336](https://github.com/hashicorp/terraform/issues/14336))
* provider/google: resource_compute_disks can now reference snapshots using the snapshot URL ([#14774](https://github.com/hashicorp/terraform/issues/14774))
* provider/heroku: Add import support for `heroku_pipeline` resource ([#14486](https://github.com/hashicorp/terraform/issues/14486))
* provider/heroku: Add import support for `heroku_pipeline_coupling` resource ([#14495](https://github.com/hashicorp/terraform/issues/14495))
* provider/heroku: Add import support for `heroku_addon` resource ([#14508](https://github.com/hashicorp/terraform/issues/14508))
* provider/openstack: Add support for all protocols in Security Group Rules ([#14307](https://github.com/hashicorp/terraform/issues/14307))
* provider/openstack: Add support for updating Subnet Allocation Pools ([#14782](https://github.com/hashicorp/terraform/issues/14782))
* provider/openstack: Enable Security Group Updates ([#14815](https://github.com/hashicorp/terraform/issues/14815))
* provider/rancher: Add member support to `rancher_environment` ([#14563](https://github.com/hashicorp/terraform/issues/14563))
* provider/rundeck: adds `description` to `command` schema in `rundeck_job` resource ([#14352](https://github.com/hashicorp/terraform/issues/14352))
* provider/scaleway: allow public_ip to be set on server resource ([#14515](https://github.com/hashicorp/terraform/issues/14515))
* provider/vsphere: Exposing moid value from vm resource ([#14793](https://github.com/hashicorp/terraform/issues/14793))

BUG FIXES:

* core: Store and verify checksums for S3 remote state to prevent fetching a stale state ([#14746](https://github.com/hashicorp/terraform/issues/14746))
* core: Allow -force-unlock of an S3 named state ([#14680](https://github.com/hashicorp/terraform/issues/14680))
* core: Fix incorrect errors when validatin nested objects ([#14784](https://github.com/hashicorp/terraform/issues/14784)] [[#14801](https://github.com/hashicorp/terraform/issues/14801))
* core: When using `-target`, any outputs that include attributes of the targeted resources are now updated ([#14186](https://github.com/hashicorp/terraform/issues/14186))
* core: Fixed 0.9.5 regression with the conditional operator `.. ? .. : ..` failing to type check with unknown/computed values ([#14454](https://github.com/hashicorp/terraform/issues/14454))
* core: Fixed 0.9 regression causing issues during refresh when adding new data resource instances using `count` ([#14098](https://github.com/hashicorp/terraform/issues/14098))
* core: Fixed crasher when populating a "splat variable" from an empty (nil) module state. ([#14526](https://github.com/hashicorp/terraform/issues/14526))
* core: fix bad Sprintf in backend migration message ([#14601](https://github.com/hashicorp/terraform/issues/14601))
* core: Addressed 0.9.5 issue with passing partially-unknown splat results through module variables, by removing the requirement to pass a redundant list level. ([#14737](https://github.com/hashicorp/terraform/issues/14737))
* provider/aws: Allow updating constraints in WAF SizeConstraintSet + no constraints ([#14661](https://github.com/hashicorp/terraform/issues/14661))
* provider/aws: Allow updating tuples in WAF ByteMatchSet + no tuples ([#14071](https://github.com/hashicorp/terraform/issues/14071))
* provider/aws: Allow updating tuples in WAF SQLInjectionMatchSet + no tuples ([#14667](https://github.com/hashicorp/terraform/issues/14667))
* provider/aws: Allow updating tuples in WAF XssMatchSet + no tuples ([#14671](https://github.com/hashicorp/terraform/issues/14671))
* provider/aws: Increase EIP update timeout ([#14381](https://github.com/hashicorp/terraform/issues/14381))
* provider/aws: Increase timeout for creating security group ([#14380](https://github.com/hashicorp/terraform/issues/14380)] [[#14724](https://github.com/hashicorp/terraform/issues/14724))
* provider/aws: Increase timeout for (dis)associating IPv6 addr to/from subnet ([#14401](https://github.com/hashicorp/terraform/issues/14401))
* provider/aws: Increase timeout for retrying creation of IAM server cert ([#14609](https://github.com/hashicorp/terraform/issues/14609))
* provider/aws: Increase timeout for deleting IGW ([#14705](https://github.com/hashicorp/terraform/issues/14705))
* provider/aws: Increase timeout for retrying creation of CW log subs ([#14722](https://github.com/hashicorp/terraform/issues/14722))
* provider/aws: Using the new time schema helper for RDS Instance lifecycle mgmt ([#14369](https://github.com/hashicorp/terraform/issues/14369))
* provider/aws: Using the timeout schema helper to make alb timeout cofigurable ([#14375](https://github.com/hashicorp/terraform/issues/14375))
* provider/aws: Refresh from state when CodePipeline Not Found ([#14431](https://github.com/hashicorp/terraform/issues/14431))
* provider/aws: Override spot_instance_requests volume_tags schema ([#14481](https://github.com/hashicorp/terraform/issues/14481))
* provider/aws: Allow Internet Gateway IPv6 routes ([#14484](https://github.com/hashicorp/terraform/issues/14484))
* provider/aws: ForceNew aws_launch_config when root_block_device changes ([#14507](https://github.com/hashicorp/terraform/issues/14507))
* provider/aws: Pass IAM Roles to codepipeline actions ([#14263](https://github.com/hashicorp/terraform/issues/14263))
* provider/aws: Create rule(s) for prefix-list-only AWS security group permissions on 'terraform import' ([#14528](https://github.com/hashicorp/terraform/issues/14528))
* provider/aws: Set aws_subnet ipv6_cidr_block to computed ([#14542](https://github.com/hashicorp/terraform/issues/14542))
* provider/aws: Change of aws_subnet ipv6 causing update failure ([#14545](https://github.com/hashicorp/terraform/issues/14545))
* provider/aws: Nothing to update in cloudformation should not result in errors ([#14463](https://github.com/hashicorp/terraform/issues/14463))
* provider/aws: Handling data migration in RDS snapshot restoring ([#14622](https://github.com/hashicorp/terraform/issues/14622))
* provider/aws: Mark cookies in `default_cache_behaviour` of cloudfront_distribution as required ([#12628](https://github.com/hashicorp/terraform/issues/12628))
* provider/aws: Fall back to old tagging mechanism for AWS gov and aws China ([#14627](https://github.com/hashicorp/terraform/issues/14627))
* provider/aws: Change AWS ssm_maintenance_window Read func ([#14665](https://github.com/hashicorp/terraform/issues/14665))
* provider/aws: Increase timeout for creation of route_table ([#14701](https://github.com/hashicorp/terraform/issues/14701))
* provider/aws: Retry ElastiCache cluster deletion when it's snapshotting ([#14700](https://github.com/hashicorp/terraform/issues/14700))
* provider/aws: Retry ECS service update on InvalidParameterException ([#14708](https://github.com/hashicorp/terraform/issues/14708))
* provider/aws: Retry IAM Role deletion on DeleteConflict ([#14707](https://github.com/hashicorp/terraform/issues/14707))
* provider/aws: Do not dereference source_Dest_check in aws_instance ([#14723](https://github.com/hashicorp/terraform/issues/14723))
* provider/aws: Add validation function for IAM Policies ([#14669](https://github.com/hashicorp/terraform/issues/14669))
* provider/aws: Fix panic on instance shutdown ([#14727](https://github.com/hashicorp/terraform/issues/14727))
* provider/aws: Handle migration when restoring db cluster from snapshot ([#14766](https://github.com/hashicorp/terraform/issues/14766))
* provider/aws: Provider ability to enable snapshotting on ElastiCache RG ([#14757](https://github.com/hashicorp/terraform/issues/14757))
* provider/cloudstack: `cloudstack_firewall` panicked when used with older (< v4.6) CloudStack versions ([#14044](https://github.com/hashicorp/terraform/issues/14044))
* provider/datadog: Allowed method on aggregator is `avg` ! `average` ([#14414](https://github.com/hashicorp/terraform/issues/14414))
* provider/digitalocean: Fix parsing of digitalocean dns records ([#14215](https://github.com/hashicorp/terraform/issues/14215))
* provider/github: Log HTTP requests and responses in DEBUG mode ([#14363](https://github.com/hashicorp/terraform/issues/14363))
* provider/github Check for potentially nil response from GitHub API client ([#14683](https://github.com/hashicorp/terraform/issues/14683))
* provider/google: Fix health check http/https defaults ([#14441](https://github.com/hashicorp/terraform/issues/14441))
* provider/google: Fix issue with GCP Cloud SQL Instance `disk_autoresize` ([#14582](https://github.com/hashicorp/terraform/issues/14582))
* provider/google: Fix crash creating Google Cloud SQL 2nd Generation replication instance ([#14373](https://github.com/hashicorp/terraform/issues/14373))
* provider/google: Disks now detach before getting deleted ([#14651](https://github.com/hashicorp/terraform/issues/14651))
* provider/google: Update `google_compute_target_pool`'s session_affinity default ([#14807](https://github.com/hashicorp/terraform/issues/14807))
* provider/heroku: Fix issue with setting correct CName in heroku_domain ([#14443](https://github.com/hashicorp/terraform/issues/14443))
* provider/opc: Correctly export `ip_address` in IP Addr Reservation ([#14543](https://github.com/hashicorp/terraform/issues/14543))
* provider/openstack: Handle Deleted Resources in Floating IP Association ([#14533](https://github.com/hashicorp/terraform/issues/14533))
* provider/openstack: Catch error during instance network parsing ([#14704](https://github.com/hashicorp/terraform/issues/14704))
* provider/vault: Prevent panic when no secret found ([#14435](https://github.com/hashicorp/terraform/issues/14435))

## 0.9.5 (May 11, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

* provider/aws: Users of aws_cloudfront_distributions with custom_origins have been broken due to changes in the AWS API requiring   `OriginReadTimeout` being set for updates. This has been fixed and will show as a change in terraform plan / apply. ([#13367](https://github.com/hashicorp/terraform/issues/13367))
* provider/aws: Users of China and Gov clouds, cannot use the new tagging of volumes created as part of aws_instances ([#14055](https://github.com/hashicorp/terraform/issues/14055))
* provider/aws: Skip tag operations on cloudwatch logs in govcloud partition. Currently not supported by Amazon. ([#12414](https://github.com/hashicorp/terraform/issues/12414))
* provider/aws: More consistent (un)quoting of long TXT/SPF `aws_route53_record`s.
   Previously we were trimming first 2 quotes and now we're (correctly) trimming first and last one.
   Depending on the use of quotes in your TXT/SPF records this may result in extra diff in plan/apply ([#14170](https://github.com/hashicorp/terraform/issues/14170))

FEATURES:

* **New Provider:** `gitlab` ([#13898](https://github.com/hashicorp/terraform/issues/13898))
* **New Resource:** `aws_emr_security_configuration` ([#14080](https://github.com/hashicorp/terraform/issues/14080))
* **New Resource:** `aws_ssm_maintenance_window` ([#14087](https://github.com/hashicorp/terraform/issues/14087))
* **New Resource:** `aws_ssm_maintenance_window_target` ([#14087](https://github.com/hashicorp/terraform/issues/14087))
* **New Resource:** `aws_ssm_maintenance_window_task` ([#14087](https://github.com/hashicorp/terraform/issues/14087))
* **New Resource:** `azurerm_sql_elasticpool` ([#14099](https://github.com/hashicorp/terraform/issues/14099))
* **New Resource:** `google_bigquery_table` ([#13743](https://github.com/hashicorp/terraform/issues/13743))
* **New Resource:** `google_compute_backend_bucket` ([#14015](https://github.com/hashicorp/terraform/issues/14015))
* **New Resource:** `google_compute_snapshot` ([#12482](https://github.com/hashicorp/terraform/issues/12482))
* **New Resource:** `heroku_app_feature` ([#14035](https://github.com/hashicorp/terraform/issues/14035))
* **New Resource:** `heroku_pipeline` ([#14078](https://github.com/hashicorp/terraform/issues/14078))
* **New Resource:** `heroku_pipeline_coupling` ([#14078](https://github.com/hashicorp/terraform/issues/14078))
* **New Resource:** `kubernetes_limit_range` ([#14285](https://github.com/hashicorp/terraform/issues/14285))
* **New Resource:** `kubernetes_resource_quota` ([#13914](https://github.com/hashicorp/terraform/issues/13914))
* **New Resource:** `vault_auth_backend` ([#10988](https://github.com/hashicorp/terraform/issues/10988))
* **New Data Source:** `aws_efs_file_system` ([#14041](https://github.com/hashicorp/terraform/issues/14041))
* **New Data Source:** `http`, for retrieving text data from generic HTTP servers ([#14270](https://github.com/hashicorp/terraform/issues/14270))
* **New Data Source:** `google_container_engine_versions`, for retrieving valid versions for clusters ([#14280](https://github.com/hashicorp/terraform/issues/14280))
* **New Interpolation Function:** `log`, for computing logarithms ([#12872](https://github.com/hashicorp/terraform/issues/12872))

IMPROVEMENTS:

* core: `sha512` and `base64sha512` interpolation functions, similar to their `sha256` equivalents. ([#14100](https://github.com/hashicorp/terraform/issues/14100))
* core: It's now possible to use the index operator `[ ]` to select a known value out of a partially-known list, such as using "splat syntax" and increasing the `count`. ([#14135](https://github.com/hashicorp/terraform/issues/14135))
* provider/aws: Add support for CustomOrigin timeouts to aws_cloudfront_distribution ([#13367](https://github.com/hashicorp/terraform/issues/13367))
* provider/aws: Add support for IAMDatabaseAuthenticationEnabled ([#14092](https://github.com/hashicorp/terraform/issues/14092))
* provider/aws: aws_dynamodb_table Add support for TimeToLive ([#14104](https://github.com/hashicorp/terraform/issues/14104))
* provider/aws: Add `security_configuration` support to `aws_emr_cluster` ([#14133](https://github.com/hashicorp/terraform/issues/14133))
* provider/aws: Add support for the tenancy placement option in `aws_spot_fleet_request` ([#14163](https://github.com/hashicorp/terraform/issues/14163))
* provider/aws: `aws_db_option_group` normalizes name to lowercase ([#14192](https://github.com/hashicorp/terraform/issues/14192), [#14366](https://github.com/hashicorp/terraform/issues/14366))
* provider/aws: Add support description to aws_iam_role ([#14208](https://github.com/hashicorp/terraform/issues/14208))
* provider/aws: Add support for SSM Documents to aws_cloudwatch_event_target ([#14067](https://github.com/hashicorp/terraform/issues/14067))
* provider/aws: add additional custom service endpoint options for CloudFormation, KMS, RDS, SNS & SQS ([#14097](https://github.com/hashicorp/terraform/issues/14097))
* provider/aws: Add ARN to security group data source ([#14245](https://github.com/hashicorp/terraform/issues/14245))
* provider/aws: Improve the wording of DynamoDB Validation error message ([#14256](https://github.com/hashicorp/terraform/issues/14256))
* provider/aws: Add support for importing Kinesis Streams ([#14278](https://github.com/hashicorp/terraform/issues/14278))
* provider/aws: Add `arn` attribute to `aws_ses_domain_identity` resource ([#14306](https://github.com/hashicorp/terraform/issues/14306))
* provider/aws: Add support for targets to aws_ssm_association ([#14246](https://github.com/hashicorp/terraform/issues/14246))
* provider/aws: native redis clustering support for elasticache ([#14317](https://github.com/hashicorp/terraform/issues/14317))
* provider/aws: Support updating `aws_waf_rule` predicates ([#14089](https://github.com/hashicorp/terraform/issues/14089))
* provider/azurerm: `azurerm_template_deployment` now supports String/Int/Boolean outputs ([#13670](https://github.com/hashicorp/terraform/issues/13670))
* provider/azurerm: Expose the Private IP Address for a Load Balancer, if available ([#13965](https://github.com/hashicorp/terraform/issues/13965))
* provider/dns: Fix data dns txt record set ([#14271](https://github.com/hashicorp/terraform/issues/14271))
* provider/dnsimple: Add support for import for dnsimple_records ([#9130](https://github.com/hashicorp/terraform/issues/9130))
* provider/dyn: Add verbose Dyn provider logs ([#14076](https://github.com/hashicorp/terraform/issues/14076))
* provider/google: Add support for networkIP in compute instance templates ([#13515](https://github.com/hashicorp/terraform/issues/13515))
* provider/google: google_dns_managed_zone is now importable ([#13824](https://github.com/hashicorp/terraform/issues/13824))
* provider/google: Add support for `compute_route` ([#14065](https://github.com/hashicorp/terraform/issues/14065))
* provider/google: Add `path` to `google_pubsub_subscription` ([#14238](https://github.com/hashicorp/terraform/issues/14238))
* provider/google: Improve Service Account by offering to recreate if missing ([#14282](https://github.com/hashicorp/terraform/issues/14282))
* provider/google: Log HTTP requests and responses in DEBUG mode ([#14281](https://github.com/hashicorp/terraform/issues/14281))
* provider/google: Add additional properties for google resource storage bucket object ([#14259](https://github.com/hashicorp/terraform/issues/14259))
* provider/google: Handle all 404 checks in read functions via the new function ([#14335](https://github.com/hashicorp/terraform/issues/14335))
* provider/heroku: import heroku_app resource ([#14248](https://github.com/hashicorp/terraform/issues/14248))
* provider/nomad: Add TLS options ([#13956](https://github.com/hashicorp/terraform/issues/13956))
* provider/triton: Add support for reading provider configuration from `TRITON_*` environment variables in addition to `SDC_*`([#14000](https://github.com/hashicorp/terraform/issues/14000))
* provider/triton: Add `cloud_config` argument to `triton_machine` resources for Linux containers ([#12840](https://github.com/hashicorp/terraform/issues/12840))
* provider/triton: Add `insecure_skip_tls_verify` ([#14077](https://github.com/hashicorp/terraform/issues/14077))

BUG FIXES:

* core: `module` blocks without names are now caught in validation, along with various other block types ([#14162](https://github.com/hashicorp/terraform/issues/14162))
* core: no longer will errors and normal log output get garbled together on Windows ([#14194](https://github.com/hashicorp/terraform/issues/14194))
* core: Avoid crash on empty TypeSet blocks ([#14305](https://github.com/hashicorp/terraform/issues/14305))
* provider/aws: Update aws_ebs_volume when attached ([#14005](https://github.com/hashicorp/terraform/issues/14005))
* provider/aws: Set aws_instance volume_tags to be Computed ([#14007](https://github.com/hashicorp/terraform/issues/14007))
* provider/aws: Fix issue getting partition for federated users ([#13992](https://github.com/hashicorp/terraform/issues/13992))
* provider/aws: aws_spot_instance_request not forcenew on volume_tags ([#14046](https://github.com/hashicorp/terraform/issues/14046))
* provider/aws: Exclude aws_instance volume tagging for China and Gov Clouds ([#14055](https://github.com/hashicorp/terraform/issues/14055))
* provider/aws: Fix source_dest_check with network_interface ([#14079](https://github.com/hashicorp/terraform/issues/14079))
* provider/aws: Fixes the bug where SNS delivery policy get always recreated ([#14064](https://github.com/hashicorp/terraform/issues/14064))
* provider/aws: Increase timeouts for Route Table retries ([#14345](https://github.com/hashicorp/terraform/issues/14345))
* provider/aws: Prevent Crash when importing aws_route53_record ([#14218](https://github.com/hashicorp/terraform/issues/14218))
* provider/aws: More consistent (un)quoting of long TXT/SPF `aws_route53_record`s ([#14170](https://github.com/hashicorp/terraform/issues/14170))
* provider/aws: Retry deletion of AWSConfig Rule on ResourceInUseException ([#14269](https://github.com/hashicorp/terraform/issues/14269))
* provider/aws: Refresh ssm document from state on 404 ([#14279](https://github.com/hashicorp/terraform/issues/14279))
* provider/aws: Allow zero-value ELB and ALB names ([#14304](https://github.com/hashicorp/terraform/issues/14304))
* provider/aws: Update the ignoring of AWS specific tags ([#14321](https://github.com/hashicorp/terraform/issues/14321))
* provider/aws: Adding IPv6 address to instance causes perpetual diff ([#14355](https://github.com/hashicorp/terraform/issues/14355))
* provider/aws: Fix SG update on instance with multiple network interfaces ([#14299](https://github.com/hashicorp/terraform/issues/14299))
* provider/azurerm: Fixing a bug in `azurerm_network_interface` ([#14365](https://github.com/hashicorp/terraform/issues/14365))
* provider/digitalocean: Prevent diffs when using IDs of images instead of slugs ([#13879](https://github.com/hashicorp/terraform/issues/13879))
* provider/fastly: Changes setting conditionals to optional ([#14103](https://github.com/hashicorp/terraform/issues/14103))
* provider/google: Ignore certain project services that can't be enabled directly via the api ([#13730](https://github.com/hashicorp/terraform/issues/13730))
* provider/google: Ability to add more than 25 project services ([#13758](https://github.com/hashicorp/terraform/issues/13758))
* provider/google: Fix compute instance panic with bad disk config ([#14169](https://github.com/hashicorp/terraform/issues/14169))
* provider/google: Handle `google_storage_bucket_object` not being found ([#14203](https://github.com/hashicorp/terraform/issues/14203))
* provider/google: Handle `google_compute_instance_group_manager` not being found ([#14190](https://github.com/hashicorp/terraform/issues/14190))
* provider/google: better visibility for compute_region_backend_service ([#14301](https://github.com/hashicorp/terraform/issues/14301))
* provider/heroku: Configure buildpacks correctly for both Org Apps and non-org Apps ([#13990](https://github.com/hashicorp/terraform/issues/13990))
* provider/heroku: Fix `heroku_cert` update of ssl cert ([#14240](https://github.com/hashicorp/terraform/issues/14240))
* provider/openstack: Handle disassociating deleted FloatingIP's from a server ([#14210](https://github.com/hashicorp/terraform/issues/14210))
* provider/postgres grant role when creating database ([#11452](https://github.com/hashicorp/terraform/issues/11452))
* provider/triton: Make triton machine deletes synchronous. ([#14368](https://github.com/hashicorp/terraform/issues/14368))
* provisioner/remote-exec: Fix panic from remote_exec provisioner ([#14134](https://github.com/hashicorp/terraform/issues/14134))

## 0.9.4 (26th April 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/template: Fix invalid MIME formatting in `template_cloudinit_config`.
   While the change itself is not breaking the data source it may be referenced
   e.g. in `aws_launch_configuration` and similar resources which are immutable
   and the formatting change will therefore trigger recreation ([#13752](https://github.com/hashicorp/terraform/issues/13752))

FEATURES:

* **New Provider:** `opc` - Oracle Public Cloud ([#13468](https://github.com/hashicorp/terraform/issues/13468))
* **New Provider:** `oneandone` ([#13633](https://github.com/hashicorp/terraform/issues/13633))
* **New Data Source:** `aws_ami_ids` ([#13844](https://github.com/hashicorp/terraform/issues/13844)] [[#13866](https://github.com/hashicorp/terraform/issues/13866))
* **New Data Source:** `aws_ebs_snapshot_ids` ([#13844](https://github.com/hashicorp/terraform/issues/13844)] [[#13866](https://github.com/hashicorp/terraform/issues/13866))
* **New Data Source:** `aws_kms_alias` ([#13669](https://github.com/hashicorp/terraform/issues/13669))
* **New Data Source:** `aws_kinesis_stream` ([#13562](https://github.com/hashicorp/terraform/issues/13562))
* **New Data Source:** `digitalocean_image` ([#13787](https://github.com/hashicorp/terraform/issues/13787))
* **New Data Source:** `google_compute_network` ([#12442](https://github.com/hashicorp/terraform/issues/12442))
* **New Data Source:** `google_compute_subnetwork` ([#12442](https://github.com/hashicorp/terraform/issues/12442))
* **New Resource:** `local_file` for creating local files (please see the docs for caveats) ([#12757](https://github.com/hashicorp/terraform/issues/12757))
* **New Resource:**  `alicloud_ess_scalinggroup` ([#13731](https://github.com/hashicorp/terraform/issues/13731))
* **New Resource:**  `alicloud_ess_scalingconfiguration` ([#13731](https://github.com/hashicorp/terraform/issues/13731))
* **New Resource:**  `alicloud_ess_scalingrule` ([#13731](https://github.com/hashicorp/terraform/issues/13731))
* **New Resource:**  `alicloud_ess_schedule` ([#13731](https://github.com/hashicorp/terraform/issues/13731))
* **New Resource:**  `alicloud_snat_entry` ([#13731](https://github.com/hashicorp/terraform/issues/13731))
* **New Resource:**  `alicloud_forward_entry` ([#13731](https://github.com/hashicorp/terraform/issues/13731))
* **New Resource:**  `aws_cognito_identity_pool` ([#13783](https://github.com/hashicorp/terraform/issues/13783))
* **New Resource:**  `aws_network_interface_attachment` ([#13861](https://github.com/hashicorp/terraform/issues/13861))
* **New Resource:**  `github_branch_protection` ([#10476](https://github.com/hashicorp/terraform/issues/10476))
* **New Resource:**  `google_bigquery_dataset` ([#13436](https://github.com/hashicorp/terraform/issues/13436))
* **New Resource:**  `heroku_space` ([#13921](https://github.com/hashicorp/terraform/issues/13921))
* **New Resource:**  `template_dir` for producing a directory from templates ([#13652](https://github.com/hashicorp/terraform/issues/13652))
* **New Interpolation Function:** `coalescelist()` ([#12537](https://github.com/hashicorp/terraform/issues/12537))


IMPROVEMENTS:

 * core: Add a `-reconfigure` flag to the `init` command, to configure a backend while ignoring any saved configuration ([#13825](https://github.com/hashicorp/terraform/issues/13825))
 * helper/schema: Disallow validation+diff suppression on computed fields ([#13878](https://github.com/hashicorp/terraform/issues/13878))
 * config: The interpolation function `cidrhost` now accepts a negative host number to count backwards from the end of the range ([#13765](https://github.com/hashicorp/terraform/issues/13765))
 * config: New interpolation function `matchkeys` for using values from one list to filter corresponding values from another list using a matching set. ([#13847](https://github.com/hashicorp/terraform/issues/13847))
 * state/remote/swift: Support Openstack request logging ([#13583](https://github.com/hashicorp/terraform/issues/13583))
 * provider/aws: Add an option to skip getting the supported EC2 platforms ([#13672](https://github.com/hashicorp/terraform/issues/13672))
 * provider/aws: Add `name_prefix` support to `aws_cloudwatch_log_group` ([#13273](https://github.com/hashicorp/terraform/issues/13273))
 * provider/aws: Add `bucket_prefix` to `aws_s3_bucket` ([#13274](https://github.com/hashicorp/terraform/issues/13274))
 * provider/aws: Add replica_source_db to the aws_db_instance datasource ([#13842](https://github.com/hashicorp/terraform/issues/13842))
 * provider/aws: Add IPv6 outputs to aws_subnet datasource ([#13841](https://github.com/hashicorp/terraform/issues/13841))
 * provider/aws: Exercise SecondaryPrivateIpAddressCount for network interface ([#10590](https://github.com/hashicorp/terraform/issues/10590))
 * provider/aws: Expose execution ARN + invoke URL for APIG deployment ([#13889](https://github.com/hashicorp/terraform/issues/13889))
 * provider/aws: Expose invoke ARN from Lambda function (for API Gateway) ([#13890](https://github.com/hashicorp/terraform/issues/13890))
 * provider/aws: Add tagging support to the 'aws_lambda_function' resource ([#13873](https://github.com/hashicorp/terraform/issues/13873))
 * provider/aws: Validate WAF metric names ([#13885](https://github.com/hashicorp/terraform/issues/13885))
 * provider/aws: Allow AWS Subnet to change IPv6 CIDR Block without ForceNew ([#13909](https://github.com/hashicorp/terraform/issues/13909))
 * provider/aws: Allow filtering of aws_subnet_ids by tags ([#13937](https://github.com/hashicorp/terraform/issues/13937))
 * provider/aws: Support aws_instance and volume tagging on creation ([#13945](https://github.com/hashicorp/terraform/issues/13945))
 * provider/aws: Add network_interface to aws_instance ([#12933](https://github.com/hashicorp/terraform/issues/12933))
 * provider/azurerm: VM Scale Sets - import support ([#13464](https://github.com/hashicorp/terraform/issues/13464))
 * provider/azurerm: Allow Azure China region support ([#13767](https://github.com/hashicorp/terraform/issues/13767))
 * provider/digitalocean: Export droplet prices ([#13720](https://github.com/hashicorp/terraform/issues/13720))
 * provider/fastly: Add support for GCS logging ([#13553](https://github.com/hashicorp/terraform/issues/13553))
 * provider/google: `google_compute_address` and `google_compute_global_address` are now importable ([#13270](https://github.com/hashicorp/terraform/issues/13270))
 * provider/google: `google_compute_network` is now importable  ([#13834](https://github.com/hashicorp/terraform/issues/13834))
 * provider/google: add attached_disk field to google_compute_instance ([#13443](https://github.com/hashicorp/terraform/issues/13443))
 * provider/heroku: Set App buildpacks from config ([#13910](https://github.com/hashicorp/terraform/issues/13910))
 * provider/heroku: Create Heroku app in a private space ([#13862](https://github.com/hashicorp/terraform/issues/13862))
 * provider/vault: `vault_generic_secret` resource can now optionally detect drift if it has appropriate access ([#11776](https://github.com/hashicorp/terraform/issues/11776))

BUG FIXES:

 * core: Prevent resource.Retry from adding untracked resources after the timeout: ([#13778](https://github.com/hashicorp/terraform/issues/13778))
 * core: Allow a schema.TypeList to be ForceNew and computed ([#13863](https://github.com/hashicorp/terraform/issues/13863))
 * core: Fix crash when refresh or apply build an invalid graph ([#13665](https://github.com/hashicorp/terraform/issues/13665))
 * core: Add the close provider/provisioner transformers back ([#13102](https://github.com/hashicorp/terraform/issues/13102))
 * core: Fix a crash condition by improving the flatmap.Expand() logic ([#13541](https://github.com/hashicorp/terraform/issues/13541))
 * provider/alicloud: Fix create PrePaid instance ([#13662](https://github.com/hashicorp/terraform/issues/13662))
 * provider/alicloud: Fix allocate public ip error ([#13268](https://github.com/hashicorp/terraform/issues/13268))
 * provider/alicloud: alicloud_security_group_rule: check ptr before use it [[#13731](https://github.com/hashicorp/terraform/issues/13731))
 * provider/alicloud: alicloud_instance: fix ecs internet_max_bandwidth_out cannot set zero bug ([#13731](https://github.com/hashicorp/terraform/issues/13731))
 * provider/aws: Allow force-destroying `aws_route53_zone` which has trailing dot ([#12421](https://github.com/hashicorp/terraform/issues/12421))
 * provider/aws: Allow GovCloud KMS ARNs to pass validation in `kms_key_id` attributes ([#13699](https://github.com/hashicorp/terraform/issues/13699))
 * provider/aws: Changing aws_opsworks_instance should ForceNew ([#13839](https://github.com/hashicorp/terraform/issues/13839))
 * provider/aws: Fix DB Parameter Group Name ([#13279](https://github.com/hashicorp/terraform/issues/13279))
 * provider/aws: Fix issue importing some Security Groups and Rules based on rule structure ([#13630](https://github.com/hashicorp/terraform/issues/13630))
 * provider/aws: Fix issue for cross account IAM role with `aws_lambda_permission` ([#13865](https://github.com/hashicorp/terraform/issues/13865))
 * provider/aws: Fix WAF IPSet descriptors removal on update ([#13766](https://github.com/hashicorp/terraform/issues/13766))
 * provider/aws: Increase default number of retries from 11 to 25 ([#13673](https://github.com/hashicorp/terraform/issues/13673))
 * provider/aws: Remove aws_vpc_dhcp_options if not found ([#13610](https://github.com/hashicorp/terraform/issues/13610))
 * provider/aws: Remove aws_network_acl_rule if not found ([#13608](https://github.com/hashicorp/terraform/issues/13608))
 * provider/aws: Use mutex & retry for WAF change operations ([#13656](https://github.com/hashicorp/terraform/issues/13656))
 * provider/aws: Adding support for ipv6 to aws_subnets needs migration ([#13876](https://github.com/hashicorp/terraform/issues/13876))
 * provider/aws: Fix validation of the `name_prefix` parameter of the `aws_alb` resource ([#13441](https://github.com/hashicorp/terraform/issues/13441))
 * provider/azurerm: azurerm_redis_cache resource missing hostname ([#13650](https://github.com/hashicorp/terraform/issues/13650))
 * provider/azurerm: Locking around Network Security Group / Subnets ([#13637](https://github.com/hashicorp/terraform/issues/13637))
 * provider/azurerm: Locking route table on subnet create/delete ([#13791](https://github.com/hashicorp/terraform/issues/13791))
 * provider/azurerm: VM's - fixes a bug where ssh_keys could contain a null entry ([#13755](https://github.com/hashicorp/terraform/issues/13755))
 * provider/azurerm: VM's - ignoring the case on the `create_option` field during Diff's ([#13933](https://github.com/hashicorp/terraform/issues/13933))
 * provider/azurerm: fixing a bug refreshing the `azurerm_redis_cache` ([#13899](https://github.com/hashicorp/terraform/issues/13899))
 * provider/fastly: Fix issue with using 0 for `default_ttl` ([#13648](https://github.com/hashicorp/terraform/issues/13648))
 * provider/google: Fix panic in GKE provisioning with addons ([#13954](https://github.com/hashicorp/terraform/issues/13954))
 * provider/fastly: Add ability to associate a healthcheck to a backend ([#13539](https://github.com/hashicorp/terraform/issues/13539))
 * provider/google: Stop setting the id when project creation fails ([#13644](https://github.com/hashicorp/terraform/issues/13644))
 * provider/google: Make ports in resource_compute_forwarding_rule ForceNew ([#13833](https://github.com/hashicorp/terraform/issues/13833))
 * provider/google: Validation fixes for forwarding rules ([#13952](https://github.com/hashicorp/terraform/issues/13952))
 * provider/ignition: Internal cache moved to global, instead per provider instance ([#13919](https://github.com/hashicorp/terraform/issues/13919))
 * provider/logentries: Refresh from state when resources not found ([#13810](https://github.com/hashicorp/terraform/issues/13810))
 * provider/newrelic: newrelic_alert_condition - `condition_scope` must be `application` or `instance` ([#12972](https://github.com/hashicorp/terraform/issues/12972))
 * provider/opc: fixed issue with unqualifying nats ([#13826](https://github.com/hashicorp/terraform/issues/13826))
 * provider/opc: Fix instance label if unset ([#13846](https://github.com/hashicorp/terraform/issues/13846))
 * provider/openstack: Fix updating Ports ([#13604](https://github.com/hashicorp/terraform/issues/13604))
 * provider/rabbitmq: Allow users without tags ([#13798](https://github.com/hashicorp/terraform/issues/13798))

## 0.9.3 (April 12, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:
 * provider/aws: Fix a critical bug in `aws_emr_cluster` in order to preserve the ordering
   of any arguments in `bootstrap_action`. Terraform will now enforce the ordering
   from the configuration. As a result, `aws_emr_cluster` resources may need to be
   recreated, as there is no API to update them in-place ([#13580](https://github.com/hashicorp/terraform/issues/13580))

FEATURES:

 * **New Resource:** `aws_api_gateway_method_settings` ([#13542](https://github.com/hashicorp/terraform/issues/13542))
 * **New Resource:** `aws_api_gateway_stage` ([#13540](https://github.com/hashicorp/terraform/issues/13540))
 * **New Resource:** `aws_iam_openid_connect_provider` ([#13456](https://github.com/hashicorp/terraform/issues/13456))
 * **New Resource:** `aws_lightsail_static_ip` ([#13175](https://github.com/hashicorp/terraform/issues/13175))
 * **New Resource:** `aws_lightsail_static_ip_attachment` ([#13207](https://github.com/hashicorp/terraform/issues/13207))
 * **New Resource:** `aws_ses_domain_identity` ([#13098](https://github.com/hashicorp/terraform/issues/13098))
 * **New Resource:** `azurerm_managed_disk` ([#12455](https://github.com/hashicorp/terraform/issues/12455))
 * **New Resource:** `kubernetes_persistent_volume` ([#13277](https://github.com/hashicorp/terraform/issues/13277))
 * **New Resource:** `kubernetes_persistent_volume_claim` ([#13527](https://github.com/hashicorp/terraform/issues/13527))
 * **New Resource:** `kubernetes_secret` ([#12960](https://github.com/hashicorp/terraform/issues/12960))
 * **New Data Source:** `aws_iam_role` ([#13213](https://github.com/hashicorp/terraform/issues/13213))

IMPROVEMENTS:

 * core: add `-lock-timeout` option, which will block and retry locks for the given duration ([#13262](https://github.com/hashicorp/terraform/issues/13262))
 * core: new `chomp` interpolation function which returns the given string with any trailing newline characters removed ([#13419](https://github.com/hashicorp/terraform/issues/13419))
 * backend/remote-state: Add support for assume role extensions to s3 backend ([#13236](https://github.com/hashicorp/terraform/issues/13236))
 * backend/remote-state: Filter extra entries from s3 environment listings ([#13596](https://github.com/hashicorp/terraform/issues/13596))
 * config: New interpolation functions `basename` and `dirname`, for file path manipulation ([#13080](https://github.com/hashicorp/terraform/issues/13080))
 * helper/resource: Allow unknown "pending" states ([#13099](https://github.com/hashicorp/terraform/issues/13099))
 * command/hook_ui: Increase max length of state IDs from 20 to 80 ([#13317](https://github.com/hashicorp/terraform/issues/13317))
 * provider/aws: Add support to set iam_role_arn on cloudformation Stack ([#12547](https://github.com/hashicorp/terraform/issues/12547))
 * provider/aws: Support priority and listener_arn update of alb_listener_rule ([#13125](https://github.com/hashicorp/terraform/issues/13125))
 * provider/aws: Deprecate roles in favour of role in iam_instance_profile ([#13130](https://github.com/hashicorp/terraform/issues/13130))
 * provider/aws: Make alb_target_group_attachment port optional ([#13139](https://github.com/hashicorp/terraform/issues/13139))
 * provider/aws: `aws_api_gateway_domain_name` `certificate_private_key` field marked as sensitive ([#13147](https://github.com/hashicorp/terraform/issues/13147))
 * provider/aws: `aws_directory_service_directory` `password` field marked as sensitive ([#13147](https://github.com/hashicorp/terraform/issues/13147))
 * provider/aws: `aws_kinesis_firehose_delivery_stream` `password` field marked as sensitive ([#13147](https://github.com/hashicorp/terraform/issues/13147))
 * provider/aws: `aws_opsworks_application` `app_source.0.password` & `ssl_configuration.0.private_key` fields marked as sensitive ([#13147](https://github.com/hashicorp/terraform/issues/13147))
 * provider/aws: `aws_opsworks_stack` `custom_cookbooks_source.0.password` field marked as sensitive ([#13147](https://github.com/hashicorp/terraform/issues/13147))
 * provider/aws: Support the ability to enable / disable ipv6 support in VPC ([#12527](https://github.com/hashicorp/terraform/issues/12527))
 * provider/aws: Added API Gateway integration update ([#13249](https://github.com/hashicorp/terraform/issues/13249))
 * provider/aws: Add `identifier` | `name_prefix` to RDS resources ([#13232](https://github.com/hashicorp/terraform/issues/13232))
 * provider/aws: Validate `aws_ecs_task_definition.container_definitions` ([#12161](https://github.com/hashicorp/terraform/issues/12161))
 * provider/aws: Update caller_identity data source ([#13092](https://github.com/hashicorp/terraform/issues/13092))
 * provider/aws: `aws_subnet_ids` data source for getting a list of subnet ids matching certain criteria ([#13188](https://github.com/hashicorp/terraform/issues/13188))
 * provider/aws: Support ip_address_type for aws_alb ([#13227](https://github.com/hashicorp/terraform/issues/13227))
 * provider/aws: Migrate `aws_dms_*` resources away from AWS waiters ([#13291](https://github.com/hashicorp/terraform/issues/13291))
 * provider/aws: Add support for treat_missing_data to cloudwatch_metric_alarm ([#13358](https://github.com/hashicorp/terraform/issues/13358))
 * provider/aws: Add support for evaluate_low_sample_count_percentiles to cloudwatch_metric_alarm ([#13371](https://github.com/hashicorp/terraform/issues/13371))
 * provider/aws: Add `name_prefix` to `aws_alb_target_group` ([#13442](https://github.com/hashicorp/terraform/issues/13442))
 * provider/aws: Add support for EMR clusters to aws_appautoscaling_target ([#13368](https://github.com/hashicorp/terraform/issues/13368))
 * provider/aws: Add import capabilities to codecommit_repository ([#13577](https://github.com/hashicorp/terraform/issues/13577))
 * provider/bitbucket: Improved error handling ([#13390](https://github.com/hashicorp/terraform/issues/13390))
 * provider/cloudstack: Do not force a new resource when updating `cloudstack_loadbalancer_rule` members ([#11786](https://github.com/hashicorp/terraform/issues/11786))
 * provider/fastly: Add support for Sumologic logging ([#12541](https://github.com/hashicorp/terraform/issues/12541))
 * provider/github: Handle the case when issue labels already exist ([#13182](https://github.com/hashicorp/terraform/issues/13182))
 * provider/google: Mark `google_container_cluster`'s `client_key` & `password` inside `master_auth` as sensitive ([#13148](https://github.com/hashicorp/terraform/issues/13148))
 * provider/google: Add node_pool field in resource_container_cluster ([#13402](https://github.com/hashicorp/terraform/issues/13402))
 * provider/kubernetes: Allow defining custom config context ([#12958](https://github.com/hashicorp/terraform/issues/12958))
 * provider/openstack: Add support for 'value_specs' options to `openstack_compute_servergroup_v2` ([#13380](https://github.com/hashicorp/terraform/issues/13380))
 * provider/statuscake: Add support for StatusCake TriggerRate field ([#13340](https://github.com/hashicorp/terraform/issues/13340))
 * provider/triton: Move to joyent/triton-go ([#13225](https://github.com/hashicorp/terraform/issues/13225))
 * provisioner/chef: Make sure we add new Chef-Vault clients as clients ([#13525](https://github.com/hashicorp/terraform/issues/13525))

BUG FIXES:

 * core: Escaped interpolation-like sequences (like `$${foo}`) now permitted in variable defaults ([#13137](https://github.com/hashicorp/terraform/issues/13137))
 * core: Fix strange issues with computed values in provider configuration that were worked around with `-input=false` ([#11264](https://github.com/hashicorp/terraform/issues/11264)], [[#13264](https://github.com/hashicorp/terraform/issues/13264))
 * core: Fix crash when providing nested maps as variable values in a `module` block ([#13343](https://github.com/hashicorp/terraform/issues/13343))
 * core: `connection` block attributes are now subject to basic validation of attribute names during validate walk ([#13400](https://github.com/hashicorp/terraform/issues/13400))
 * provider/aws: Add Support for maintenance_window and back_window to rds_cluster_instance ([#13134](https://github.com/hashicorp/terraform/issues/13134))
 * provider/aws: Increase timeout for AMI registration ([#13159](https://github.com/hashicorp/terraform/issues/13159))
 * provider/aws: Increase timeouts for ELB ([#13161](https://github.com/hashicorp/terraform/issues/13161))
 * provider/aws: `volume_type` of `aws_elasticsearch_domain.0.ebs_options` marked as `Computed` which prevents spurious diffs ([#13160](https://github.com/hashicorp/terraform/issues/13160))
 * provider/aws: Don't set DBName on `aws_db_instance` from snapshot ([#13140](https://github.com/hashicorp/terraform/issues/13140))
 * provider/aws: Add DiffSuppression to aws_ecs_service placement_strategies ([#13220](https://github.com/hashicorp/terraform/issues/13220))
 * provider/aws: Refresh aws_alb_target_group stickiness on manual updates ([#13199](https://github.com/hashicorp/terraform/issues/13199))
 * provider/aws: Preserve default retain_on_delete in cloudfront import ([#13209](https://github.com/hashicorp/terraform/issues/13209))
 * provider/aws: Refresh aws_alb_target_group tags ([#13200](https://github.com/hashicorp/terraform/issues/13200))
 * provider/aws: Set aws_vpn_connection to recreate when in deleted state ([#13204](https://github.com/hashicorp/terraform/issues/13204))
 * provider/aws: Wait for aws_opsworks_instance to be running when it's specified ([#13218](https://github.com/hashicorp/terraform/issues/13218))
 * provider/aws: Handle `aws_lambda_function` missing s3 key error ([#10960](https://github.com/hashicorp/terraform/issues/10960))
 * provider/aws: Set stickiness to computed in alb_target_group ([#13278](https://github.com/hashicorp/terraform/issues/13278))
 * provider/aws: Increase timeout for deploying `cloudfront_distribution` from 40 to 70 mins ([#13319](https://github.com/hashicorp/terraform/issues/13319))
 * provider/aws: Increase AMI retry timeouts ([#13324](https://github.com/hashicorp/terraform/issues/13324))
 * provider/aws: Increase subnet deletion timeout ([#13356](https://github.com/hashicorp/terraform/issues/13356))
 * provider/aws: Increase launch_configuration creation timeout ([#13357](https://github.com/hashicorp/terraform/issues/13357))
 * provider/aws: Increase Beanstalk env 'ready' timeout ([#13359](https://github.com/hashicorp/terraform/issues/13359))
 * provider/aws: Raise timeout for deleting APIG REST API ([#13414](https://github.com/hashicorp/terraform/issues/13414))
 * provider/aws: Raise timeout for attaching/detaching VPN Gateway ([#13457](https://github.com/hashicorp/terraform/issues/13457))
 * provider/aws: Recreate opsworks_stack on change of service_role_arn ([#13325](https://github.com/hashicorp/terraform/issues/13325))
 * provider/aws: Fix KMS Key reading with Exists method ([#13348](https://github.com/hashicorp/terraform/issues/13348))
 * provider/aws: Fix DynamoDB issues about GSIs indexes ([#13256](https://github.com/hashicorp/terraform/issues/13256))
 * provider/aws: Fix `aws_s3_bucket` drift detection of logging options ([#13281](https://github.com/hashicorp/terraform/issues/13281))
 * provider/aws: Update ElasticTranscoderPreset to have default for MaxFrameRate ([#13422](https://github.com/hashicorp/terraform/issues/13422))
 * provider/aws: Fix aws_ami_launch_permission refresh when AMI disappears ([#13469](https://github.com/hashicorp/terraform/issues/13469))
 * provider/aws: Add support for updating SSM documents ([#13491](https://github.com/hashicorp/terraform/issues/13491))
 * provider/aws: Fix panic on nil route configs ([#13548](https://github.com/hashicorp/terraform/issues/13548))
 * provider/azurerm: Network Security Group - ignoring protocol casing at Import time ([#13153](https://github.com/hashicorp/terraform/issues/13153))
 * provider/azurerm: Fix crash when importing Local Network Gateways ([#13261](https://github.com/hashicorp/terraform/issues/13261))
 * provider/azurerm: Defaulting the value of `duplicate_detection_history_time_window` for `azurerm_servicebus_topic` ([#13223](https://github.com/hashicorp/terraform/issues/13223))
 * provider/azurerm: Event Hubs making the Location field idempotent ([#13570](https://github.com/hashicorp/terraform/issues/13570))
 * provider/bitbucket: Fixed issue where provider would fail with an "EOF" error on some operations ([#13390](https://github.com/hashicorp/terraform/issues/13390))
 * provider/dnsimple: Handle 404 on DNSimple records ([#13131](https://github.com/hashicorp/terraform/issues/13131))
 * provider/kubernetes: Use PATCH to update namespace ([#13114](https://github.com/hashicorp/terraform/issues/13114))
 * provider/ns1: No splitting answer on SPF records. ([#13260](https://github.com/hashicorp/terraform/issues/13260))
 * provider/openstack: Refresh volume_attachment from state if NotFound ([#13342](https://github.com/hashicorp/terraform/issues/13342))
 * provider/openstack: Add SOFT_DELETED to delete status ([#13444](https://github.com/hashicorp/terraform/issues/13444))
 * provider/profitbricks: Changed output type of ips variable of ip_block ProfitBricks resource ([#13290](https://github.com/hashicorp/terraform/issues/13290))
 * provider/template: Fix panic in cloudinit config ([#13581](https://github.com/hashicorp/terraform/issues/13581))

## 0.9.2 (March 28, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/openstack: Port Fixed IPs are able to be read again using the original numerical notation. However, Fixed IP configurations which are obtaining addresses via DHCP must now use the `all_fixed_ips` attribute to reference the returned IP address.
 * Environment names must be safe to use as a URL path segment without escaping, and is enforced by the CLI.

FEATURES:

 * **New Resource:**  `alicloud_db_instance` ([#12913](https://github.com/hashicorp/terraform/issues/12913))
 * **New Resource:**  `aws_api_gateway_usage_plan` ([#12542](https://github.com/hashicorp/terraform/issues/12542))
 * **New Resource:**  `aws_api_gateway_usage_plan_key` ([#12851](https://github.com/hashicorp/terraform/issues/12851))
 * **New Resource:**  `github_repository_webhook` ([#12924](https://github.com/hashicorp/terraform/issues/12924))
 * **New Resource:**  `random_pet` ([#12903](https://github.com/hashicorp/terraform/issues/12903))
 * **New Interpolation:** `substr` ([#12870](https://github.com/hashicorp/terraform/issues/12870))
 * **S3 Environments:** The S3 remote state backend now supports named environments

IMPROVEMENTS:

 * core: fix interpolation error when referencing computed values from an `aws_instance` `cidr_block` ([#13046](https://github.com/hashicorp/terraform/issues/13046))
 * core: fix `ignore_changes` causing fields to be removed during apply ([#12897](https://github.com/hashicorp/terraform/issues/12897))
 * core: add `-force-copy` option to `terraform init` to supress prompts for copying state ([#12939](https://github.com/hashicorp/terraform/issues/12939))
 * helper/acctest: Add NewSSHKeyPair function ([#12894](https://github.com/hashicorp/terraform/issues/12894))
 * provider/alicloud: simplify validators ([#12982](https://github.com/hashicorp/terraform/issues/12982))
 * provider/aws: Added support for EMR AutoScalingRole ([#12823](https://github.com/hashicorp/terraform/issues/12823))
 * provider/aws: Add `name_prefix` to `aws_autoscaling_group` and `aws_elb` resources ([#12629](https://github.com/hashicorp/terraform/issues/12629))
 * provider/aws: Updated default configuration manager version in `aws_opsworks_stack` ([#12979](https://github.com/hashicorp/terraform/issues/12979))
 * provider/aws: Added aws_api_gateway_api_key value attribute ([#9462](https://github.com/hashicorp/terraform/issues/9462))
 * provider/aws: Allow aws_alb subnets to change ([#12850](https://github.com/hashicorp/terraform/issues/12850))
 * provider/aws: Support Attachment of ALB Target Groups to Autoscaling Groups ([#12855](https://github.com/hashicorp/terraform/issues/12855))
 * provider/aws: Support Import of iam_server_certificate ([#13065](https://github.com/hashicorp/terraform/issues/13065))
 * provider/azurerm: Add support for setting the primary network interface ([#11290](https://github.com/hashicorp/terraform/issues/11290))
 * provider/cloudstack: Add `zone_id` to `cloudstack_ipaddress` resource ([#11306](https://github.com/hashicorp/terraform/issues/11306))
 * provider/consul: Add support for basic auth to the provider ([#12679](https://github.com/hashicorp/terraform/issues/12679))
 * provider/digitalocean: Support disk only resize ([#13059](https://github.com/hashicorp/terraform/issues/13059))
 * provider/dnsimple: Allow dnsimple_record.priority attribute to be set ([#12843](https://github.com/hashicorp/terraform/issues/12843))
 * provider/google: Add support for service_account, metadata, and image_type fields in GKE cluster config ([#12743](https://github.com/hashicorp/terraform/issues/12743))
 * provider/google: Add local ssd count support for container clusters ([#12281](https://github.com/hashicorp/terraform/issues/12281))
 * provider/ignition: ignition_filesystem, explicit option to create the filesystem ([#12980](https://github.com/hashicorp/terraform/issues/12980))
 * provider/kubernetes: Internal K8S annotations are ignored in `config_map` ([#12945](https://github.com/hashicorp/terraform/issues/12945))
 * provider/ns1: Ensure provider checks for credentials ([#12920](https://github.com/hashicorp/terraform/issues/12920))
 * provider/openstack: Adding Timeouts to Blockstorage Resources ([#12862](https://github.com/hashicorp/terraform/issues/12862))
 * provider/openstack: Adding Timeouts to FWaaS v1 Resources ([#12863](https://github.com/hashicorp/terraform/issues/12863))
 * provider/openstack: Adding Timeouts to Image v2 and LBaaS v2 Resources ([#12865](https://github.com/hashicorp/terraform/issues/12865))
 * provider/openstack: Adding Timeouts to Network Resources ([#12866](https://github.com/hashicorp/terraform/issues/12866))
 * provider/openstack: Adding Timeouts to LBaaS v1 Resources ([#12867](https://github.com/hashicorp/terraform/issues/12867))
 * provider/openstack: Deprecating Instance Volume attribute ([#13062](https://github.com/hashicorp/terraform/issues/13062))
 * provider/openstack: Decprecating Instance Floating IP attribute ([#13063](https://github.com/hashicorp/terraform/issues/13063))
 * provider/openstack: Don't log the catalog ([#13075](https://github.com/hashicorp/terraform/issues/13075))
 * provider/openstack: Handle 409/500 Response on Pool Create ([#13074](https://github.com/hashicorp/terraform/issues/13074))
 * provider/pagerduty: Validate credentials ([#12854](https://github.com/hashicorp/terraform/issues/12854))
 * provider/openstack: Adding all_metadata attribute ([#13061](https://github.com/hashicorp/terraform/issues/13061))
 * provider/profitbricks: Handling missing resources ([#13053](https://github.com/hashicorp/terraform/issues/13053))

BUG FIXES:

 * core: Remove legacy remote state configuration on state migration. This fixes errors when saving plans. ([#12888](https://github.com/hashicorp/terraform/issues/12888))
 * provider/arukas: Default timeout for launching container increased to 15mins (was 10mins) ([#12849](https://github.com/hashicorp/terraform/issues/12849))
 * provider/aws: Fix flattened cloudfront lambda function associations to be a set not a slice ([#11984](https://github.com/hashicorp/terraform/issues/11984))
 * provider/aws: Consider ACTIVE as pending state during ECS svc deletion ([#12986](https://github.com/hashicorp/terraform/issues/12986))
 * provider/aws: Deprecate the usage of Api Gateway Key Stages in favor of Usage Plans ([#12883](https://github.com/hashicorp/terraform/issues/12883))
 * provider/aws: prevent panic in resourceAwsSsmDocumentRead ([#12891](https://github.com/hashicorp/terraform/issues/12891))
 * provider/aws: Prevent panic when setting AWS CodeBuild Source to state ([#12915](https://github.com/hashicorp/terraform/issues/12915))
 * provider/aws: Only call replace Iam Instance Profile on existing machines ([#12922](https://github.com/hashicorp/terraform/issues/12922))
 * provider/aws: Increase AWS AMI Destroy timeout ([#12943](https://github.com/hashicorp/terraform/issues/12943))
 * provider/aws: Set aws_vpc ipv6 for associated only ([#12899](https://github.com/hashicorp/terraform/issues/12899))
 * provider/aws: Fix AWS ECS placement strategy spread fields ([#12998](https://github.com/hashicorp/terraform/issues/12998))
 * provider/aws: Specify that aws_network_acl_rule requires a cidr block ([#13013](https://github.com/hashicorp/terraform/issues/13013))
 * provider/aws: aws_network_acl_rule treat all and -1 for protocol the same ([#13049](https://github.com/hashicorp/terraform/issues/13049))
 * provider/aws: Only allow 1 value in alb_listener_rule condition ([#13051](https://github.com/hashicorp/terraform/issues/13051))
 * provider/aws: Correct handling of network ACL default IPv6 ingress/egress rules ([#12835](https://github.com/hashicorp/terraform/issues/12835))
 * provider/aws: aws_ses_receipt_rule: fix off-by-one errors ([#12961](https://github.com/hashicorp/terraform/issues/12961))
 * provider/aws: Fix issue upgrading to Terraform v0.9+ with AWS OpsWorks Stacks ([#13024](https://github.com/hashicorp/terraform/issues/13024))
 * provider/fastly: Fix issue importing Fastly Services with Backends ([#12538](https://github.com/hashicorp/terraform/issues/12538))
 * provider/google: turn compute_instance_group.instances into a set ([#12790](https://github.com/hashicorp/terraform/issues/12790))
 * provider/mysql: recreate user/grant if user/grant got deleted manually ([#12791](https://github.com/hashicorp/terraform/issues/12791))
 * provider/openstack: Fix monitor_id typo in LBaaS v1 Pool ([#13069](https://github.com/hashicorp/terraform/issues/13069))
 * provider/openstack: Resolve issues with Port Fixed IPs ([#13056](https://github.com/hashicorp/terraform/issues/13056))
 * provider/rancher: error when no api_url is provided ([#13086](https://github.com/hashicorp/terraform/issues/13086))
 * provider/scaleway: work around parallel request limitation ([#13045](https://github.com/hashicorp/terraform/issues/13045))

## 0.9.1 (March 17, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/pagerduty: the deprecated `name_regex` field has been removed from vendor data source ([#12396](https://github.com/hashicorp/terraform/issues/12396))

FEATURES:

 * **New Provider:** `kubernetes` ([#12372](https://github.com/hashicorp/terraform/issues/12372))
 * **New Resource:** `kubernetes_namespace` ([#12372](https://github.com/hashicorp/terraform/issues/12372))
 * **New Resource:** `kubernetes_config_map` ([#12753](https://github.com/hashicorp/terraform/issues/12753))
 * **New Data Source:** `dns_a_record_set` ([#12744](https://github.com/hashicorp/terraform/issues/12744))
 * **New Data Source:** `dns_cname_record_set` ([#12744](https://github.com/hashicorp/terraform/issues/12744))
 * **New Data Source:** `dns_txt_record_set` ([#12744](https://github.com/hashicorp/terraform/issues/12744))

IMPROVEMENTS:

 * command/init: `-backend-config` accepts `key=value` pairs
 * provider/aws: Improved error when failing to get S3 tags ([#12759](https://github.com/hashicorp/terraform/issues/12759))
 * provider/aws: Validate CIDR Blocks in SG and SG rule resources ([#12765](https://github.com/hashicorp/terraform/issues/12765))
 * provider/aws: Add KMS key tag support ([#12243](https://github.com/hashicorp/terraform/issues/12243))
 * provider/aws: Allow `name_prefix` to be used with various IAM resources ([#12658](https://github.com/hashicorp/terraform/issues/12658))
 * provider/openstack: Add timeout support for Compute resources ([#12794](https://github.com/hashicorp/terraform/issues/12794))
 * provider/scaleway: expose public IPv6 information on scaleway_server ([#12748](https://github.com/hashicorp/terraform/issues/12748))

BUG FIXES:

 * core: Fix panic when an undefined module is reference ([#12793](https://github.com/hashicorp/terraform/issues/12793))
 * core: Fix regression from 0.8.x when using a data source in a module ([#12837](https://github.com/hashicorp/terraform/issues/12837))
 * command/apply: Applies from plans with backends set will reuse the backend rather than local ([#12785](https://github.com/hashicorp/terraform/issues/12785))
 * command/init: Changing only `-backend-config` detects changes and reconfigures ([#12776](https://github.com/hashicorp/terraform/issues/12776))
 * command/init: Fix legacy backend init error that could occur when upgrading ([#12818](https://github.com/hashicorp/terraform/issues/12818))
 * command/push: Detect local state and error properly ([#12773](https://github.com/hashicorp/terraform/issues/12773))
 * command/refresh: Allow empty and non-existent state ([#12777](https://github.com/hashicorp/terraform/issues/12777))
 * provider/aws: Get the aws_lambda_function attributes when there are great than 50 versions of a function ([#11745](https://github.com/hashicorp/terraform/issues/11745))
 * provider/aws: Correctly check for nil cidr_block in aws_network_acl ([#12735](https://github.com/hashicorp/terraform/issues/12735))
 * provider/aws: Stop setting weight property on route53_record read ([#12756](https://github.com/hashicorp/terraform/issues/12756))
 * provider/google: Fix the Google provider asking for account_file input on every run ([#12729](https://github.com/hashicorp/terraform/issues/12729))
 * provider/profitbricks: Prevent panic on profitbricks volume ([#12819](https://github.com/hashicorp/terraform/issues/12819))


## 0.9.0 (March 15, 2017)

**This is the complete 0.8.8 to 0.9 CHANGELOG. Below this section we also have a 0.9.0-beta2 to 0.9.0 final CHANGELOG.**

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/aws: `aws_codebuild_project` renamed `timeout` to `build_timeout` ([#12503](https://github.com/hashicorp/terraform/issues/12503))
 * provider/azurem: `azurerm_virtual_machine` and `azurerm_virtual_machine_scale_set` now store has of custom_data not all custom_data ([#12214](https://github.com/hashicorp/terraform/issues/12214))
 * provider/azurerm: scale_sets `os_profile_master_password` now marked as sensitive
 * provider/azurerm: sql_server `administrator_login_password` now marked as sensitive
 * provider/dnsimple: Provider has been upgraded to APIv2 therefore, you will need to use the APIv2 auth token
 * provider/google: storage buckets have been updated with the new storage classes. The old classes will continue working as before, but should be migrated as soon as possible, as there's no guarantee they'll continue working forever. ([#12044](https://github.com/hashicorp/terraform/issues/12044))
 * provider/google: compute_instance, compute_instance_template, and compute_disk all have a subtly changed logic when specifying an image family as the image; in 0.8.x they would pin to the latest image in the family when the resource is created; in 0.9.x they pass the family to the API and use its behaviour. New input formats are also supported. ([#12223](https://github.com/hashicorp/terraform/issues/12223))
 * provider/google: removed the unused and deprecated region field from google_compute_backend_service ([#12663](https://github.com/hashicorp/terraform/issues/12663))
 * provider/google: removed the deprecated account_file field for the Google Cloud provider ([#12668](https://github.com/hashicorp/terraform/issues/12668))
 * provider/google: removed the deprecated fields from google_project ([#12659](https://github.com/hashicorp/terraform/issues/12659))

FEATURES:

 * **Remote Backends:** This is a successor to "remote state" and includes
   file-based configuration, an improved setup process (just run `terraform init`),
   no more local caching of remote state, and more. ([#11286](https://github.com/hashicorp/terraform/issues/11286))
 * **Destroy Provisioners:** Provisioners can now be configured to run
   on resource destruction. ([#11329](https://github.com/hashicorp/terraform/issues/11329))
 * **State Locking:** State will be automatically locked when supported by the backend.
   Backends supporting locking in this release are Local, S3 (via DynamoDB), and Consul. ([#11187](https://github.com/hashicorp/terraform/issues/11187))
 * **State Environments:** You can now create named "environments" for states. This allows you to manage distinct infrastructure resources from the same configuration.
 * **New Provider:**  `Circonus` ([#12578](https://github.com/hashicorp/terraform/issues/12578))
 * **New Data Source:**  `openstack_networking_network_v2` ([#12304](https://github.com/hashicorp/terraform/issues/12304))
 * **New Resource:**  `aws_iam_account_alias` ([#12648](https://github.com/hashicorp/terraform/issues/12648))
 * **New Resource:**  `datadog_downtime` ([#10994](https://github.com/hashicorp/terraform/issues/10994))
 * **New Resource:**  `ns1_notifylist` ([#12373](https://github.com/hashicorp/terraform/issues/12373))
 * **New Resource:**  `google_container_node_pool` ([#11802](https://github.com/hashicorp/terraform/issues/11802))
 * **New Resource:**  `rancher_certificate` ([#12717](https://github.com/hashicorp/terraform/issues/12717))
 * **New Resource:**  `rancher_host` ([#11545](https://github.com/hashicorp/terraform/issues/11545))
 * helper/schema: Added Timeouts to allow Provider/Resource developers to expose configurable timeouts for actions ([#12311](https://github.com/hashicorp/terraform/issues/12311))

IMPROVEMENTS:

 * core: Data source values can now be used as part of a `count` calculation. ([#11482](https://github.com/hashicorp/terraform/issues/11482))
 * core: "terraformrc" can contain env var references with $FOO ([#11929](https://github.com/hashicorp/terraform/issues/11929))
 * core: report all errors encountered during config validation ([#12383](https://github.com/hashicorp/terraform/issues/12383))
 * command: CLI args can be specified via env vars. Specify `TF_CLI_ARGS` or `TF_CLI_ARGS_name` (where name is the name of a command) to specify additional CLI args ([#11922](https://github.com/hashicorp/terraform/issues/11922))
 * command/init: previous behavior is retained, but init now also configures
   the new remote backends as well as downloads modules. It is the single
   command to initialize a new or existing Terraform configuration.
 * command: Display resource state ID in refresh/plan/destroy output ([#12261](https://github.com/hashicorp/terraform/issues/12261))
 * provider/aws: AWS Lambda DeadLetterConfig support ([#12188](https://github.com/hashicorp/terraform/issues/12188))
 * provider/aws: Return errors from Elastic Beanstalk ([#12425](https://github.com/hashicorp/terraform/issues/12425))
 * provider/aws: Set aws_db_cluster to snapshot by default ([#11668](https://github.com/hashicorp/terraform/issues/11668))
 * provider/aws: Enable final snapshots for aws_rds_cluster by default ([#11694](https://github.com/hashicorp/terraform/issues/11694))
 * provider/aws: Enable snapshotting by default on aws_redshift_cluster ([#11695](https://github.com/hashicorp/terraform/issues/11695))
 * provider/aws: Add support for ACM certificates to `api_gateway_domain_name` ([#12592](https://github.com/hashicorp/terraform/issues/12592))
 * provider/aws: Add support for IPv6 to aws\_security\_group\_rule ([#12645](https://github.com/hashicorp/terraform/issues/12645))
 * provider/aws: Add IPv6 Support to aws\_route\_table ([#12640](https://github.com/hashicorp/terraform/issues/12640))
 * provider/aws: Add support for IPv6 to aws\_network\_acl\_rule ([#12644](https://github.com/hashicorp/terraform/issues/12644))
 * provider/aws: Add support for IPv6 to aws\_default\_route\_table ([#12642](https://github.com/hashicorp/terraform/issues/12642))
 * provider/aws: Add support for IPv6 to aws\_network\_acl ([#12641](https://github.com/hashicorp/terraform/issues/12641))
 * provider/aws: Add support for IPv6 in aws\_route ([#12639](https://github.com/hashicorp/terraform/issues/12639))
 * provider/aws: Add support for IPv6 to aws\_security\_group ([#12655](https://github.com/hashicorp/terraform/issues/12655))
 * provider/aws: Add replace\_unhealthy\_instances to spot\_fleet\_request ([#12681](https://github.com/hashicorp/terraform/issues/12681))
 * provider/aws: Remove restriction on running aws\_opsworks\_* on us-east-1 ([#12688](https://github.com/hashicorp/terraform/issues/12688))
 * provider/aws: Improve error message on S3 Bucket Object deletion ([#12712](https://github.com/hashicorp/terraform/issues/12712))
 * provider/aws: Add log message about if changes are being applied now or later ([#12691](https://github.com/hashicorp/terraform/issues/12691))
 * provider/azurerm: Mark the azurerm_scale_set machine password as sensitive ([#11982](https://github.com/hashicorp/terraform/issues/11982))
 * provider/azurerm: Mark the azurerm_sql_server admin password as sensitive ([#12004](https://github.com/hashicorp/terraform/issues/12004))
 * provider/azurerm: Add support for managed availability sets. ([#12532](https://github.com/hashicorp/terraform/issues/12532))
 * provider/azurerm: Add support for extensions on virtual machine scale sets ([#12124](https://github.com/hashicorp/terraform/issues/12124))
 * provider/dnsimple: Upgrade DNSimple provider to API v2 ([#10760](https://github.com/hashicorp/terraform/issues/10760))
 * provider/docker: added support for linux capabilities ([#12045](https://github.com/hashicorp/terraform/issues/12045))
 * provider/fastly: Add Fastly SSL validation fields ([#12578](https://github.com/hashicorp/terraform/issues/12578))
 * provider/ignition: Migrate all of the igition resources to data sources ([#11851](https://github.com/hashicorp/terraform/issues/11851))
 * provider/openstack: Set Availability Zone in Instances ([#12610](https://github.com/hashicorp/terraform/issues/12610))
 * provider/openstack: Force Deletion of Instances ([#12689](https://github.com/hashicorp/terraform/issues/12689))
 * provider/rancher: Better comparison of compose files ([#12561](https://github.com/hashicorp/terraform/issues/12561))
 * provider/azurerm: store only hash of `azurerm_virtual_machine` and `azurerm_virtual_machine_scale_set` custom_data - reduces size of state ([#12214](https://github.com/hashicorp/terraform/issues/12214))
 * provider/vault: read vault token from `~/.vault-token` as a fallback for the
   `VAULT_TOKEN` environment variable. ([#11529](https://github.com/hashicorp/terraform/issues/11529))
 * provisioners: All provisioners now respond very quickly to interrupts for
   fast cancellation. ([#10934](https://github.com/hashicorp/terraform/issues/10934))

BUG FIXES:

 * core: targeting will remove untargeted providers ([#12050](https://github.com/hashicorp/terraform/issues/12050))
 * core: doing a map lookup in a resource config with a computed set no longer crashes ([#12210](https://github.com/hashicorp/terraform/issues/12210))
 * provider/aws: Fixes issue for aws_lb_ssl_negotiation_policy of already deleted ELB ([#12360](https://github.com/hashicorp/terraform/issues/12360))
 * provider/aws: Populate the iam_instance_profile uniqueId ([#12449](https://github.com/hashicorp/terraform/issues/12449))
 * provider/aws: Only send iops when creating io1 devices ([#12392](https://github.com/hashicorp/terraform/issues/12392))
 * provider/aws: Fix spurious aws_spot_fleet_request diffs ([#12437](https://github.com/hashicorp/terraform/issues/12437))
 * provider/aws: Changing volumes in ECS task definition should force new revision ([#11403](https://github.com/hashicorp/terraform/issues/11403))
 * provider/aws: Ignore whitespace in json diff for aws_dms_replication_task options ([#12380](https://github.com/hashicorp/terraform/issues/12380))
 * provider/aws: Check spot instance is running before trying to attach volumes ([#12459](https://github.com/hashicorp/terraform/issues/12459))
 * provider/aws: Add the IPV6 cidr block to the vpc datasource ([#12529](https://github.com/hashicorp/terraform/issues/12529))
 * provider/aws: Error on trying to recreate an existing customer gateway ([#12501](https://github.com/hashicorp/terraform/issues/12501))
 * provider/aws: Prevent aws_dms_replication_task panic ([#12539](https://github.com/hashicorp/terraform/issues/12539))
 * provider/aws: output the task definition name when errors occur during refresh ([#12609](https://github.com/hashicorp/terraform/issues/12609))
 * provider/aws: Refresh iam saml provider from state on 404 ([#12602](https://github.com/hashicorp/terraform/issues/12602))
 * provider/aws: Add address, port, hosted_zone_id and endpoint for aws_db_instance datasource ([#12623](https://github.com/hashicorp/terraform/issues/12623))
 * provider/aws: Allow recreation of `aws_opsworks_user_profile` when the `user_arn` is changed ([#12595](https://github.com/hashicorp/terraform/issues/12595))
 * provider/aws: Guard clause to prevent panic on ELB connectionSettings ([#12685](https://github.com/hashicorp/terraform/issues/12685))
 * provider/azurerm: bug fix to prevent crashes during azurerm_container_service provisioning ([#12516](https://github.com/hashicorp/terraform/issues/12516))
 * provider/cobbler: Fix Profile Repos ([#12452](https://github.com/hashicorp/terraform/issues/12452))
 * provider/datadog: Update to datadog_monitor to use default values ([#12497](https://github.com/hashicorp/terraform/issues/12497))
 * provider/datadog: Default notify_no_data on datadog_monitor to false ([#11903](https://github.com/hashicorp/terraform/issues/11903))
 * provider/google: Correct the incorrect instance group manager URL returned from GKE ([#4336](https://github.com/hashicorp/terraform/issues/4336))
 * provider/google: Fix a plan/apply cycle in IAM policies ([#12387](https://github.com/hashicorp/terraform/issues/12387))
 * provider/google: Fix a plan/apply cycle in forwarding rules when only a single port is specified ([#12662](https://github.com/hashicorp/terraform/issues/12662))
 * provider/google: Minor correction : "Deleting disk" message in Delete method ([#12521](https://github.com/hashicorp/terraform/issues/12521))
 * provider/mysql: Avoid crash on un-interpolated provider cfg ([#12391](https://github.com/hashicorp/terraform/issues/12391))
 * provider/ns1: Fix incorrect schema (causing crash) for 'ns1_user.notify' ([#12721](https://github.com/hashicorp/terraform/issues/12721))
 * provider/openstack: Handle cases where volumes are disabled ([#12374](https://github.com/hashicorp/terraform/issues/12374))
 * provider/openstack: Toggle Creation of Default Security Group Rules ([#12119](https://github.com/hashicorp/terraform/issues/12119))
 * provider/openstack: Change Port fixed_ip to a Set ([#12613](https://github.com/hashicorp/terraform/issues/12613))
 * provider/openstack: Add network_id to Network data source ([#12615](https://github.com/hashicorp/terraform/issues/12615))
 * provider/openstack: Check for ErrDefault500 when creating/deleting pool member ([#12664](https://github.com/hashicorp/terraform/issues/12664))
 * provider/rancher: Apply the set value for finish_upgrade to set to prevent recurring plans ([#12545](https://github.com/hashicorp/terraform/issues/12545))
 * provider/scaleway: work around API concurrency issue ([#12707](https://github.com/hashicorp/terraform/issues/12707))
 * provider/statuscake: use default status code list when updating test ([#12375](https://github.com/hashicorp/terraform/issues/12375))

## 0.9.0 from 0.9.0-beta2 (March 15, 2017)

**This only includes changes from 0.9.0-beta2 to 0.9.0 final. The section above has the complete 0.8.x to 0.9.0 CHANGELOG.**

FEATURES:

 * **New Provider:**  `Circonus` ([#12578](https://github.com/hashicorp/terraform/issues/12578))

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/aws: `aws_codebuild_project` renamed `timeout` to `build_timeout` ([#12503](https://github.com/hashicorp/terraform/issues/12503))
 * provider/azurem: `azurerm_virtual_machine` and `azurerm_virtual_machine_scale_set` now store has of custom_data not all custom_data ([#12214](https://github.com/hashicorp/terraform/issues/12214))
 * provider/google: compute_instance, compute_instance_template, and compute_disk all have a subtly changed logic when specifying an image family as the image; in 0.8.x they would pin to the latest image in the family when the resource is created; in 0.9.x they pass the family to the API and use its behaviour. New input formats are also supported. ([#12223](https://github.com/hashicorp/terraform/issues/12223))
 * provider/google: removed the unused and deprecated region field from google_compute_backend_service ([#12663](https://github.com/hashicorp/terraform/issues/12663))
 * provider/google: removed the deprecated account_file field for the Google Cloud provider ([#12668](https://github.com/hashicorp/terraform/issues/12668))
 * provider/google: removed the deprecated fields from google_project ([#12659](https://github.com/hashicorp/terraform/issues/12659))

IMPROVEMENTS:

 * provider/azurerm: store only hash of `azurerm_virtual_machine` and `azurerm_virtual_machine_scale_set` custom_data - reduces size of state ([#12214](https://github.com/hashicorp/terraform/issues/12214))
 * report all errors encountered during config validation ([#12383](https://github.com/hashicorp/terraform/issues/12383))

BUG FIXES:

 * provider/google: Correct the incorrect instance group manager URL returned from GKE ([#4336](https://github.com/hashicorp/terraform/issues/4336))
 * provider/google: Fix a plan/apply cycle in IAM policies ([#12387](https://github.com/hashicorp/terraform/issues/12387))
 * provider/google: Fix a plan/apply cycle in forwarding rules when only a single port is specified ([#12662](https://github.com/hashicorp/terraform/issues/12662))

## 0.9.0-beta2 (March 2, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/azurerm: scale_sets `os_profile_master_password` now marked as sensitive
 * provider/azurerm: sql_server `administrator_login_password` now marked as sensitive
 * provider/google: storage buckets have been updated with the new storage classes. The old classes will continue working as before, but should be migrated as soon as possible, as there's no guarantee they'll continue working forever. ([#12044](https://github.com/hashicorp/terraform/issues/12044))
 * provider/dnsimple: Provider has been upgraded to APIv2 therefore, you will need to use the APIv2 auth token

FEATURES:

 * **State Environments:** You can now create named "environments" for states. This allows you to manage distinct infrastructure resources from the same configuration.
 * helper/schema: Added Timeouts to allow Provider/Resource developers to expose configurable timeouts for actions ([#12311](https://github.com/hashicorp/terraform/issues/12311))

IMPROVEMENTS:

 * core: "terraformrc" can contain env var references with $FOO ([#11929](https://github.com/hashicorp/terraform/issues/11929))
 * command: Display resource state ID in refresh/plan/destroy output ([#12261](https://github.com/hashicorp/terraform/issues/12261))
 * provider/aws: AWS Lambda DeadLetterConfig support ([#12188](https://github.com/hashicorp/terraform/issues/12188))
 * provider/azurerm: Mark the azurerm_scale_set machine password as sensitive ([#11982](https://github.com/hashicorp/terraform/issues/11982))
 * provider/azurerm: Mark the azurerm_sql_server admin password as sensitive ([#12004](https://github.com/hashicorp/terraform/issues/12004))
 * provider/dnsimple: Upgrade DNSimple provider to API v2 ([#10760](https://github.com/hashicorp/terraform/issues/10760))

BUG FIXES:

 * core: targeting will remove untargeted providers ([#12050](https://github.com/hashicorp/terraform/issues/12050))
 * core: doing a map lookup in a resource config with a computed set no longer crashes ([#12210](https://github.com/hashicorp/terraform/issues/12210))

0.9.0-beta1 FIXES:

 * core: backends are validated to not contain interpolations ([#12067](https://github.com/hashicorp/terraform/issues/12067))
 * core: fix local state locking on Windows ([#12059](https://github.com/hashicorp/terraform/issues/12059))
 * core: destroy provisioners dependent on module variables work ([#12063](https://github.com/hashicorp/terraform/issues/12063))
 * core: resource destruction happens after dependent resources' destroy provisioners ([#12063](https://github.com/hashicorp/terraform/issues/12063))
 * core: invalid resource attribute interpolation in a destroy provisioner errors ([#12063](https://github.com/hashicorp/terraform/issues/12063))
 * core: legacy backend loading of Consul now works properly ([#12320](https://github.com/hashicorp/terraform/issues/12320))
 * command/init: allow unsetting a backend properly ([#11988](https://github.com/hashicorp/terraform/issues/11988))
 * command/apply: fix crash that could happen with an empty directory ([#11989](https://github.com/hashicorp/terraform/issues/11989))
 * command/refresh: fix crash when no configs were in the pwd ([#12178](https://github.com/hashicorp/terraform/issues/12178))
 * command/{state,taint}: work properly with backend state ([#12155](https://github.com/hashicorp/terraform/issues/12155))
 * providers/terraform: remote state data source works with new backends ([#12173](https://github.com/hashicorp/terraform/issues/12173))

## 0.9.0-beta1 (February 15, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * Once an environment is updated to use the new "remote backend" feature
   (from a prior remote state), it cannot be used with prior Terraform versions.
   Remote backends themselves are fully backwards compatible with prior
   Terraform versions.
 * provider/aws: `aws_db_instance` now defaults to making a final snapshot on delete
 * provider/aws: `aws_rds_cluster` now defaults to making a final snapshot on delete
 * provider/aws: `aws_redshift_cluster` now defaults to making a final snapshot on delete
 * provider/aws: Deprecated fields `kinesis_endpoint` & `dynamodb_endpoint` were removed. Use `kinesis` & `dynamodb` inside the `endpoints` block instead. ([#11778](https://github.com/hashicorp/terraform/issues/11778))
 * provider/datadog: `datadog_monitor` now defaults `notify_no_data` to `false` as per the datadog API

FEATURES:

 * **Remote Backends:** This is a successor to "remote state" and includes
   file-based configuration, an improved setup process (just run `terraform init`),
   no more local caching of remote state, and more. ([#11286](https://github.com/hashicorp/terraform/issues/11286))
 * **Destroy Provisioners:** Provisioners can now be configured to run
   on resource destruction. ([#11329](https://github.com/hashicorp/terraform/issues/11329))
 * **State Locking:** State will be automatically locked when supported by the backend.
   Backends supporting locking in this release are Local, S3 (via DynamoDB), and Consul. ([#11187](https://github.com/hashicorp/terraform/issues/11187))

IMPROVEMENTS:

 * core: Data source values can now be used as part of a `count` calculation. ([#11482](https://github.com/hashicorp/terraform/issues/11482))
 * command: CLI args can be specified via env vars. Specify `TF_CLI_ARGS` or `TF_CLI_ARGS_name` (where name is the name of a command) to specify additional CLI args ([#11922](https://github.com/hashicorp/terraform/issues/11922))
 * command/init: previous behavior is retained, but init now also configures
   the new remote backends as well as downloads modules. It is the single
   command to initialize a new or existing Terraform configuration.
 * provisioners: All provisioners now respond very quickly to interrupts for
   fast cancellation. ([#10934](https://github.com/hashicorp/terraform/issues/10934))
 * provider/aws: Set aws_db_cluster to snapshot by default ([#11668](https://github.com/hashicorp/terraform/issues/11668))
 * provider/aws: Enable final snapshots for aws_rds_cluster by default ([#11694](https://github.com/hashicorp/terraform/issues/11694))
 * provider/aws: Enable snapshotting by default on aws_redshift_cluster ([#11695](https://github.com/hashicorp/terraform/issues/11695))
 * provider/vault: read vault token from `~/.vault-token` as a fallback for the
   `VAULT_TOKEN` environment variable. ([#11529](https://github.com/hashicorp/terraform/issues/11529))

BUG FIXES:

 * provider/datadog: Default notify_no_data on datadog_monitor to false ([#11903](https://github.com/hashicorp/terraform/issues/11903))

## 0.8.8 (March 2, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:
 * provider/aws: Potential breaking change for `root_block_device` ([#12379](https://github.com/hashicorp/terraform/issues/12379))

FEATURES:

 * **New Provider:**  `spotinst` ([#5001](https://github.com/hashicorp/terraform/issues/5001))
 * **New Interpolation:**  `slice` ([#9729](https://github.com/hashicorp/terraform/issues/9729))
 * **New Data Source:**  `aws_sns_topic` ([#11752](https://github.com/hashicorp/terraform/issues/11752))
 * **New Data Source:**  `openstack_images_image_v2` ([#12097](https://github.com/hashicorp/terraform/issues/12097))
 * **New Resource:**  `aws_elastic_beanstalk_application_version` ([#5770](https://github.com/hashicorp/terraform/issues/5770))
 * **New Resource:**  `aws_cloudwatch_log_destination` ([#11940](https://github.com/hashicorp/terraform/issues/11940))
 * **New Resource:**  `aws_cloudwatch_log_destination_policy` ([#11940](https://github.com/hashicorp/terraform/issues/11940))
 * **New Resource:**  `aws_codepipeline` ([#11814](https://github.com/hashicorp/terraform/issues/11814))
 * **New Resource:**  `aws_egress_only_internet_gateway` ([#10538](https://github.com/hashicorp/terraform/issues/10538))
 * **New Resource:**  `datadog_user`  ([#12268](https://github.com/hashicorp/terraform/issues/12268))
 * **New Resource:**  `digitalocean_loadbalancer` ([#12077](https://github.com/hashicorp/terraform/issues/12077))
 * **New Resource:**  `openstack_images_image_v2` ([#11942](https://github.com/hashicorp/terraform/issues/11942))
 * **New Resource:**  `openstack_compute_floatingip_associate_v2` ([#12190](https://github.com/hashicorp/terraform/issues/12190))

IMPROVEMENTS:

 * provider/aws: Add support for AWS EBS Elastic Volumes ([#11981](https://github.com/hashicorp/terraform/issues/11981))
 * provider/aws: Allow aws_instances to be resized rather than forcing a new instance ([#11998](https://github.com/hashicorp/terraform/issues/11998))
 * provider/aws: Report bucket name in S3 Error message ([#12122](https://github.com/hashicorp/terraform/issues/12122))
 * provider/aws: Implement IPV6 Support for ec2 / VPC ([#10538](https://github.com/hashicorp/terraform/issues/10538))
 * provider/aws: Add support for import of aws_elasticsearch_domain ([#12330](https://github.com/hashicorp/terraform/issues/12330))
 * provider/aws: improve redshift cluster validation ([#12313](https://github.com/hashicorp/terraform/issues/12313))
 * provider/aws: Support IAM role attachment and replacement for existing EC2 instance ([#11852](https://github.com/hashicorp/terraform/issues/11852))
 * provider/azurerm: Auto base64encode virtual_machine custom data ([#12164](https://github.com/hashicorp/terraform/issues/12164))
 * provider/datadog: add support for new host delay to the datadog_monitor resource ([#11975](https://github.com/hashicorp/terraform/issues/11975))
 * provider/datadog: Upgrade to Datadog API v2 ([#12098](https://github.com/hashicorp/terraform/issues/12098))
 * provider/fastly: Make Backends optional if used in VCL ([#12025](https://github.com/hashicorp/terraform/issues/12025))
 * provider/fastly: Add support for custom `response_object` ([#12032](https://github.com/hashicorp/terraform/issues/12032))
 * provider/google: Add support for maintenance window in `sql_database_instance` ([#12042](https://github.com/hashicorp/terraform/issues/12042))
 * provider/google: google_project supports billing account ([#11653](https://github.com/hashicorp/terraform/issues/11653))
 * provider/openstack: Don't allow floating IP and port ([#12099](https://github.com/hashicorp/terraform/issues/12099))
 * provider/openstack: Enable HTTP Logging ([#12089](https://github.com/hashicorp/terraform/issues/12089))
 * provider/openstack: Add Additional Targets for LBaaS v1 Member ([#12266](https://github.com/hashicorp/terraform/issues/12266))
 * provider/openstack: Redesign openstack_blockstorage_volume_attach_v2 ([#12071](https://github.com/hashicorp/terraform/issues/12071))
 * provider/pagerduty: Import support for service integrations ([#12141](https://github.com/hashicorp/terraform/issues/12141))
 * provider/pagerduty: Updated implementation of pagerduty_vendor & pagerduty_service_integration ([#12357](https://github.com/hashicorp/terraform/issues/12357))
 * provider/random_id: Add prefix attribute ([#12016](https://github.com/hashicorp/terraform/issues/12016))
 * provider/statuscake: Add support for Port in statuscake_test ([#11966](https://github.com/hashicorp/terraform/issues/11966))

BUG FIXES:

 * core: Fix a hang that could occur at the end of a Terraform command with custom plugins used ([#12048](https://github.com/hashicorp/terraform/issues/12048))
 * command/fmt: Fix incorrect formatting with single line object following complex object ([#12049](https://github.com/hashicorp/terraform/issues/12049))
 * command/state: `-backup` flags work with `mv` and `rm` ([#12156](https://github.com/hashicorp/terraform/issues/12156))
 * provider/aws: add bucket name to delete error notification ([#11952](https://github.com/hashicorp/terraform/issues/11952))
 * provider/aws: Use proper Set for source.Auth in resource_aws_codebuild_project ([#11741](https://github.com/hashicorp/terraform/issues/11741))
 * provider/aws: aws_ecs_service should output service name along with err ([#12072](https://github.com/hashicorp/terraform/issues/12072))
 * provider/aws: Add VRRP to allowed protocols in network ACL rules ([#12107](https://github.com/hashicorp/terraform/issues/12107))
 * provider/aws: Add owner_account option to aws_redshift_cluster ([#12062](https://github.com/hashicorp/terraform/issues/12062))
 * provider/aws: Update of inspector_assessment_target should use ARN not Name ([#12115](https://github.com/hashicorp/terraform/issues/12115))
 * provider/aws: Fix the panic in ssm_association with parameters ([#12215](https://github.com/hashicorp/terraform/issues/12215))
 * provider/aws: Fix update of environment_variable in codebuild_project ([#12169](https://github.com/hashicorp/terraform/issues/12169))
 * provider/aws: Refresh aws_autoscaling_schedule from state when autoscaling_group not found ([#12312](https://github.com/hashicorp/terraform/issues/12312))
 * provider/aws: No longer ForceNew resource on lambda_function runtime update ([#12329](https://github.com/hashicorp/terraform/issues/12329))
 * provider/aws: reading multiple pages of aws_efs_file_system tags ([#12328](https://github.com/hashicorp/terraform/issues/12328))
 * provider/aws: Refresh cloudwatch log subscription filter on 404 ([#12333](https://github.com/hashicorp/terraform/issues/12333))
 * provider/aws: more details on which s3 bucket had an error ([#12314](https://github.com/hashicorp/terraform/issues/12314))
 * provider/azurerm: Ignore case on protocol and allocation types ([#12176](https://github.com/hashicorp/terraform/issues/12176))
 * provider/cloudflare: add validation for proxied record types ([#11993](https://github.com/hashicorp/terraform/issues/11993))
 * provider/datadog: Adding default values to datadog_monitor ([#12168](https://github.com/hashicorp/terraform/issues/12168))
 * provider/google: make local_traffic_selector computed ([#11631](https://github.com/hashicorp/terraform/issues/11631))
 * provider/google: Write the raw disk encryption key in the state file to avoid diffs on plan ([#12068](https://github.com/hashicorp/terraform/issues/12068))
 * provider/google: fix url map test and update logic ([#12317](https://github.com/hashicorp/terraform/issues/12317))
 * provider/openstack: Rename provider to loadbalancer_provider ([#12239](https://github.com/hashicorp/terraform/issues/12239))
 * provider/pagerduty: Setting incident_urgency_rule as optional ([#12211](https://github.com/hashicorp/terraform/issues/12211))
 * provider/profitbricks: Fixing how primary_nic is added to profitbricks server ([#12197](https://github.com/hashicorp/terraform/issues/12197))
 * state/azure: add environment option for non-public cloud usage ([#12364](https://github.com/hashicorp/terraform/issues/12364))

## 0.8.7 (February 15, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/aws: `kinesis_endpoint` & `dynamodb_endpoint` fields in the provider schema were deprecated in favour of `kinesis` & `dynamodb` inside the `endpoints` block. Deprecated fields will be removed in 0.9 ([#11768](https://github.com/hashicorp/terraform/issues/11768))

FEATURES:

 * **New Interpolation:**  `slice` ([#9729](https://github.com/hashicorp/terraform/issues/9729))
 * **New Provider:**  `arukas` ([#11171](https://github.com/hashicorp/terraform/issues/11171))
 * **New Data Source:**  `aws_db_instance` ([#11717](https://github.com/hashicorp/terraform/issues/11717))
 * **New Data Source:**  `aws_vpn_gateway` ([#11886](https://github.com/hashicorp/terraform/issues/11886))
 * **New Data Source:**  `consul_agent_self`, `consul_catalog_service`, `consul_catalog_services`, `consul_catalog_nodes` ([#11729](https://github.com/hashicorp/terraform/pull/11729))
 * **New Data Source:**  `google_compute_zones` ([#11954](https://github.com/hashicorp/terraform/issues/11954))
 * **New Resource:**  `aws_elasticsearch_domain_policy` ([#8648](https://github.com/hashicorp/terraform/issues/8648))
 * **New Resource:**  `aws_vpc_peering_connection_accepter` ([#11505](https://github.com/hashicorp/terraform/issues/11505))
 * **New Resource:**  `aws_config_config_rule` ([#5850](https://github.com/hashicorp/terraform/issues/5850))
 * **New Resource:**  `aws_config_configuration_recorder` ([#5850](https://github.com/hashicorp/terraform/issues/5850))
 * **New Resource:**  `aws_config_configuration_recorder_status` ([#5850](https://github.com/hashicorp/terraform/issues/5850))
 * **New Resource:**  `aws_config_delivery_channel` ([#5850](https://github.com/hashicorp/terraform/issues/5850))
 * **New Resource:**  `azurerm_container_service` ([#10820](https://github.com/hashicorp/terraform/issues/10820))
 * **New Resource:**  `vault_policy` ([#10980](https://github.com/hashicorp/terraform/issues/10980))

IMPROVEMENTS:

 * provider/aws: Update aws_ssm_document to include `document_type`, `latest_version` and `default_version` ([#11671](https://github.com/hashicorp/terraform/issues/11671))
 * provider/aws: Support import of aws_opsworks_instance ([#11783](https://github.com/hashicorp/terraform/issues/11783))
 * provider/aws Add S3 bucket object tag support ([#11344](https://github.com/hashicorp/terraform/issues/11344))
 * provider/aws: Add validation for aws_iam_role ([#11915](https://github.com/hashicorp/terraform/issues/11915))
 * provider/fastly Allows for conditional settings across fastly ([#11843](https://github.com/hashicorp/terraform/issues/11843))
 * provider/openstack: Allow OpenStack SSL certs + keys to take path or content ([#10271](https://github.com/hashicorp/terraform/issues/10271))
 * provider/pagerduty: Add support for `incident_urgency_rule`, `support_hours` and `scheduled_actions` to `pagerduty_service` ([#11856](https://github.com/hashicorp/terraform/issues/11856))
 * provider/rancher: parse Rancher client cli.json config file ([#11658](https://github.com/hashicorp/terraform/issues/11658))
 * provider/vault: Use Vault api.DefaultConfig() ([#11523](https://github.com/hashicorp/terraform/issues/11523))

Bug FIXES:

 * core: resources that depend on create-before-destroy resources don't create cycles ([#11753](https://github.com/hashicorp/terraform/issues/11753))
 * core: create-before-destroy resources with a count > 1 create proper edges ([#11753](https://github.com/hashicorp/terraform/issues/11753))
 * core: fix "diffs didn't match issue" for removing or empty collections that force new ([#11732](https://github.com/hashicorp/terraform/issues/11732))
 * core: module sources ended in archive extensions without a "." won't be treated as archives ([#11438](https://github.com/hashicorp/terraform/issues/11438))
 * core: destroy ordering of resources within modules is correct ([#11765](https://github.com/hashicorp/terraform/issues/11765))
 * core: Fix crash if count interpolates into a non-int ([#11864](https://github.com/hashicorp/terraform/issues/11864))
 * core: Targeting a module will properly exclude untargeted module outputs ([#11921](https://github.com/hashicorp/terraform/issues/11921))
 * state/remote/s3: Fix Bug with Assume Role for Federated IAM Account ([#10067](https://github.com/hashicorp/terraform/issues/10067))
 * provider/aws: Fix security_group_rule resource timeout errors ([#11809](https://github.com/hashicorp/terraform/issues/11809))
 * provider/aws: Fix diff suppress function for aws_db_instance ([#11909](https://github.com/hashicorp/terraform/issues/11909))
 * provider/aws: Fix default values for AMI volume size ([#11842](https://github.com/hashicorp/terraform/issues/11842))
 * provider/aws: Fix aws_db_event_subscription import ([#11744](https://github.com/hashicorp/terraform/issues/11744))
 * provider/aws: Respect 400 returned from AWS API on RDS Cluster termination ([#11795](https://github.com/hashicorp/terraform/issues/11795))
 * provider/aws: Raise the codebuild_project create timeout ([#11777](https://github.com/hashicorp/terraform/issues/11777))
 * provider/aws: Make aws_dms_endpoint database_name optional ([#11792](https://github.com/hashicorp/terraform/issues/11792))
 * provider/aws: Bump Create and Delete timeouts to 60 mins on directory_service ([#11793](https://github.com/hashicorp/terraform/issues/11793))
 * provider/aws: aws_codecommit_trigger fix typo that causes serialization to fail when events is non-empty ([#11839](https://github.com/hashicorp/terraform/issues/11839))
 * provider/aws: Fix bug to allow update of maintenance_window in elasticache_replication_group ([#11850](https://github.com/hashicorp/terraform/issues/11850))
 * provider/azurerm: Don't push an empty set of ssh keys to virtual machine or they cannot be ammended ([#11804](https://github.com/hashicorp/terraform/issues/11804))
 * provider/azurerm: Refresh from state when VM Extension Resource not found ([#11894](https://github.com/hashicorp/terraform/issues/11894))
 * provider/cloudstack: Ensure consistent hashes of `cloudstack_port_forward` forward items. ([#11546](https://github.com/hashicorp/terraform/issues/11546))
 * provider/google: set additional_zones to computed and disallow the original zone from appearing in the list ([#11650](https://github.com/hashicorp/terraform/issues/11650))
 * provider/google: set subnetwork_project to computed ([#11646](https://github.com/hashicorp/terraform/issues/11646))
 * provider/openstack BlockStorage v1 availability_zone Fix ([#11949](https://github.com/hashicorp/terraform/issues/11949))

## 0.8.6 (07 February 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/aws: `aws_appautoscaling_policy` no longer has default values for `scalable_dimension` and `service_namespace`


FEATURES:

 * **New Data Source:**  `aws_kms_secret` ([#11460](https://github.com/hashicorp/terraform/issues/11460))
 * **New Data Source:**  `aws_ecs_task_definition` ([#8509](https://github.com/hashicorp/terraform/issues/8509))
 * **New Data Source:**  `aws_ecs_cluster` ([#11558](https://github.com/hashicorp/terraform/issues/11558))
 * **New Data Source:**  `aws_partition` ([#11675](https://github.com/hashicorp/terraform/issues/11675))
 * **New Data Source:**  `pagerduty_escalation_policy` ([#11616](https://github.com/hashicorp/terraform/issues/11616))
 * **New Data Source:**  `pagerduty_schedule` ([#11614](https://github.com/hashicorp/terraform/issues/11614))
 * **New Data Source:**  `profitbricks_datacenter` ([#11520](https://github.com/hashicorp/terraform/issues/11520))
 * **New Data Source:**  `profitbricks_location` ([#11520](https://github.com/hashicorp/terraform/issues/11520))
 * **New Data Source:**  `profitbricks_image` ([#11520](https://github.com/hashicorp/terraform/issues/11520))
 * **New Resource:**  `aws_sfn_activity` ([#11420](https://github.com/hashicorp/terraform/issues/11420))
 * **New Resource:**  `aws_sfn_state_machine` ([#11420](https://github.com/hashicorp/terraform/issues/11420))
 * **New Resource:**  `aws_codebuild_project` ([#11560](https://github.com/hashicorp/terraform/issues/11560))
 * **New Resource:**  `aws_dms_certificate` ([#11122](https://github.com/hashicorp/terraform/issues/11122))
 * **New Resource:**  `aws_dms_endpoint` ([#11122](https://github.com/hashicorp/terraform/issues/11122))
 * **New Resource:**  `aws_dms_replication_instance` ([#11122](https://github.com/hashicorp/terraform/issues/11122))
 * **New Resource:**  `aws_dms_replication_subnet_group` ([#11122](https://github.com/hashicorp/terraform/issues/11122))
 * **New Resource:**  `aws_dms_replication_subnet_group` ([#11122](https://github.com/hashicorp/terraform/issues/11122))
 * **New Resource:**  `pagerduty_addon` ([#11620](https://github.com/hashicorp/terraform/issues/11620))


IMPROVEMENTS:

 * core: Interaction with Atlas now supports the `ATLAS_TLS_NOVERIFY` environment variable ([#11576](https://github.com/hashicorp/terraform/issues/11576))
 * provider/aws: Add EBS Volume support for EMR Instance Groups ([#11411](https://github.com/hashicorp/terraform/issues/11411))
 * provider/aws: Add support for policy to AWS provider assume_role ([#11501](https://github.com/hashicorp/terraform/issues/11501))
 * provider/aws: Add support for more sns_topic_subscription parameters on import command ([#10408](https://github.com/hashicorp/terraform/issues/10408))
 * provider/aws: Add support for Sever Side Encryption with default S3 KMS key to `aws_s3_bucket_object` ([#11261](https://github.com/hashicorp/terraform/issues/11261))
 * provider/aws: Add support for Cross Region RDS Cluster Replica ([#11428](https://github.com/hashicorp/terraform/issues/11428))
 * provider/aws: Add sensitive attribute in master_password ([#11584](https://github.com/hashicorp/terraform/issues/11584))
 * provider/aws: Application Auto Scaling now supports scaling an Amazon EC2 Spot fleet ([#8697](https://github.com/hashicorp/terraform/issues/8697))
 * provider/aws: Add tag support to DynamoDb tables ([#11617](https://github.com/hashicorp/terraform/issues/11617))
 * provider/aws: Provide the certificate ID in the aws data source ([#11693](https://github.com/hashicorp/terraform/issues/11693))
 * provider/aws: Wait for instance_profile creation to complete ([#11678](https://github.com/hashicorp/terraform/issues/11678))
 * provider/azurerm: Add support for scale sets overprovision ([#11516](https://github.com/hashicorp/terraform/issues/11516))
 * provider/azurerm: support import for load balancer and sub resources ([#11610](https://github.com/hashicorp/terraform/issues/11610))
 * provider/fastly: Adds papertrail logging ([#11491](https://github.com/hashicorp/terraform/issues/11491))
 * provider/fastly: Adds format_version for s3logging ([#11725](https://github.com/hashicorp/terraform/issues/11725))
 * provider/fastly: Adds healthcheck service ([#11709](https://github.com/hashicorp/terraform/issues/11709))
 * provider/google: allow instance group managers in region other than project ([#11294](https://github.com/hashicorp/terraform/issues/11294))
 * provider/google: Add second generation disk specification options ([#11571](https://github.com/hashicorp/terraform/issues/11571))
 * provider/google: remote_traffic_selector for google_compute_vpn_tunnel ([#11020](https://github.com/hashicorp/terraform/issues/11020))
 * provider/nomad: Update jobspec dependency to allow parsing parameterized nomad jobfiles ([#11691](https://github.com/hashicorp/terraform/issues/11691))
 * provider/google: No default root user for SQL ([#11590](https://github.com/hashicorp/terraform/issues/11590))
 * provider/opsgenie: Descriptions for Teams ([#11391](https://github.com/hashicorp/terraform/issues/11391))
 * provider/rancher: rancher_registration_token add image parameter ([#11551](https://github.com/hashicorp/terraform/issues/11551))
 * provider/rancher: allow for importing resources using environment ID to target ([#11688](https://github.com/hashicorp/terraform/issues/11688))

BUG FIXES:

 * core: Remove missed subfields when parent list is removed ([#11498](https://github.com/hashicorp/terraform/issues/11498))
 * command/fmt: Trailing blocks of comments at the end of files are formatted properly ([#11585](https://github.com/hashicorp/terraform/issues/11585))
 * provider/aws: Fix issue with `path` not updated when modifying AWS API Gateway Resource ([#11443](https://github.com/hashicorp/terraform/issues/11443))
 * provider/aws: Fix AWS Lambda Qualifier Regexp for `aws_lambda_permission` ([#11383](https://github.com/hashicorp/terraform/issues/11383))
 * provider/aws: allow destroy of LB stickiness policy with missing LB ([#11462](https://github.com/hashicorp/terraform/issues/11462))
 * provider/aws: ECS Placement constraints fix ([#11475](https://github.com/hashicorp/terraform/issues/11475))
 * provider/aws: retry kms_key CreateKey if arn in policy not yet seen ([#11509](https://github.com/hashicorp/terraform/issues/11509))
 * provider/aws: Fix ALB Listener Rule Import ([#1174](https://github.com/hashicorp/terraform/issues/1174))
 * provider/aws: Fix issue with ECS Placement Strat. and type casing ([#11565](https://github.com/hashicorp/terraform/issues/11565))
 * provider/aws: aws_route53_record import error processing ([#11603](https://github.com/hashicorp/terraform/issues/11603))
 * provider/aws: Fix panic in aws_rds_cluster missing parameter error message ([#11600](https://github.com/hashicorp/terraform/issues/11600))
 * provider/aws: Succeed creating aws_volume_attachment if identical attachment exists ([#11060](https://github.com/hashicorp/terraform/issues/11060))
 * provider/aws: Guard against panic in aws_vpc_endpoint_association ([#11613](https://github.com/hashicorp/terraform/issues/11613))
 * provider/aws: Allow root volume size changes in aws_instance ([#11619](https://github.com/hashicorp/terraform/issues/11619))
 * provider/aws: Fix spot instance request block device configs ([#11649](https://github.com/hashicorp/terraform/issues/11649))
 * provider/aws: Fix validation issues for onceAWeek and onceADay validation functions ([#11679](https://github.com/hashicorp/terraform/issues/11679))
 * provider/aws: Return route_table_id from aws_route_table data source ([#11703](https://github.com/hashicorp/terraform/issues/11703))
 * provider/aws: validate aws_alb_target_group name is less than 32 characters ([#11699](https://github.com/hashicorp/terraform/issues/11699))
 * provider/azurerm: Scale Sets Load balancer pools should not be computed ([#11516](https://github.com/hashicorp/terraform/issues/11516))
 * provider/azurerm: Scale Sets ip configuration handling and update support for load balancer backend pools. ([#11516](https://github.com/hashicorp/terraform/issues/11516))
 * provider/azurerm: check if lb sub resources exist when reading ([#11553](https://github.com/hashicorp/terraform/issues/11553))
 * provider/google: Fix master_instance_name to prevent slave rebuilds ([#11477](https://github.com/hashicorp/terraform/issues/11477))
 * provider/google: Refresh google_compute_instance machine_type on read ([#11645](https://github.com/hashicorp/terraform/issues/11645))
 * provider/google: Added forceNew on accessConfig in google_compute_instance_template ([#11548](https://github.com/hashicorp/terraform/issues/11548))
 * provider/ignition: Allow to add authorized keys without user creation ([#11406](https://github.com/hashicorp/terraform/issues/11406))
 * provider/ignition: mount and path are mutually exclusive ([#11409](https://github.com/hashicorp/terraform/issues/11409))
 * provider/ns1: Fix "use_client_subnet" in ns1_record ([#11368](https://github.com/hashicorp/terraform/issues/11368))
 * provider/openstack: Remove Default Security Group Rules on Create ([#11466](https://github.com/hashicorp/terraform/issues/11466))
 * provider/pagerduty: Allow timeouts to be disabled (pagerduty_service) ([#11483](https://github.com/hashicorp/terraform/issues/11483))
 * provider/rancher: Use environment specific client for accessing resources ([#11503](https://github.com/hashicorp/terraform/issues/11503))
 * provider/rancher: Refresh rancher stack from state on delete ([#11539](https://github.com/hashicorp/terraform/issues/11539))
 * provider/rancher: Refresh rancher token and registry from state on not found ([#11543](https://github.com/hashicorp/terraform/issues/11543))
 * provider/rancher: return error when Rancher template not found ([#11544](https://github.com/hashicorp/terraform/issues/11544))
 * provider/rancher: rancher_stack set docker_compose and rancher_compose ([#11550](https://github.com/hashicorp/terraform/issues/11550))
 * provider/rancher: Handle deleted/purged resources from Rancher ([#11607](https://github.com/hashicorp/terraform/issues/11607))
 * provider/statuscake: Remove computed from statuscake_test timeout parameter ([#11541](https://github.com/hashicorp/terraform/issues/11541))
 * provider/vsphere: vSphere virtual machine don't ignore VM power on errors ([#11604](https://github.com/hashicorp/terraform/issues/11604))
 * provisioner/remote-exec: Revert change in 0.8.5 that treated each line as a script since that doesn't work for stateful scripts. ([#11692](https://github.com/hashicorp/terraform/issues/11692))
 * provisioner/chef: Attributes JSON coming from computed source validates ([#11502](https://github.com/hashicorp/terraform/issues/11502))

## 0.8.5 (26 January 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/aws: We no longer prefix an ECR repository address with `https://`
 * provider/google: `google_project` has undergone significant changes. Existing configs and state should continue to work as they always have, but new configs and state will exhibit some new behaviour, including actually creating and deleting projects, instead of just referencing them. See https://www.terraform.io/docs/providers/google/r/google_project.html for more details.

FEATURES:

 * **New Data Source:**  `aws_autoscaling_groups` ([#11303](https://github.com/hashicorp/terraform/issues/11303))
 * **New Data Source:**  `aws_elb_hosted_zone_id ` ([#11027](https://github.com/hashicorp/terraform/issues/11027))
 * **New Data Source:**  `aws_instance` ([#11272](https://github.com/hashicorp/terraform/issues/11272))
 * **New Data Source:**  `aws_canonical_user_id` ([#11332](https://github.com/hashicorp/terraform/issues/11332))
 * **New Data Source:**  `aws_vpc_endpoint` ([#11323](https://github.com/hashicorp/terraform/issues/11323))
 * **New Provider:**  `profitbricks` ([#7943](https://github.com/hashicorp/terraform/issues/7943))
 * **New Provider:**  `alicloud` ([#11235](https://github.com/hashicorp/terraform/issues/11235))
 * **New Provider:**  `ns1` ([#10782](https://github.com/hashicorp/terraform/issues/10782))
 * **New Resource:**  `aws_inspector_assessment_target` ([#11217](https://github.com/hashicorp/terraform/issues/11217))
 * **New Resource:**  `aws_inspector_assessment_template` ([#11217](https://github.com/hashicorp/terraform/issues/11217))
 * **New Resource:**  `aws_inspector_resource_group` ([#11217](https://github.com/hashicorp/terraform/issues/11217))
 * **New Resource:**  `google_project_iam_policy` ([#10425](https://github.com/hashicorp/terraform/issues/10425))
 * **New Resource:**  `google_project_services` ([#10425](https://github.com/hashicorp/terraform/issues/10425))
 * **New Interpolation Function:**  `pathexpand()` ([#11277](https://github.com/hashicorp/terraform/issues/11277))

IMPROVEMENTS:

 * command/fmt: Single line objects (such as `variable "foo" {}`) aren't separated by newlines
 * provider/aws: Add 'route_table_id' to route_table data source ([#11157](https://github.com/hashicorp/terraform/pull/11157))
 * provider/aws: Add Support for aws_cloudwatch_metric_alarm extended statistic ([#11193](https://github.com/hashicorp/terraform/issues/11193))
 * provider/aws: Make the type of a route53_record modifiable without recreating the resource ([#11164](https://github.com/hashicorp/terraform/issues/11164))
 * provider/aws: Add Placement Strategy to aws_ecs_service resource ([#11201](https://github.com/hashicorp/terraform/issues/11201))
 * provider/aws: Add support for placement_constraint to aws_ecs_service ([#11242](https://github.com/hashicorp/terraform/issues/11242))
 * provider/aws: allow ALB target group stickiness to be enabled/disabled ([#11251](https://github.com/hashicorp/terraform/issues/11251))
 * provider/aws: ALBs now wait for provisioning to complete before proceeding ([#11333](https://github.com/hashicorp/terraform/issues/11333))
 * provider/aws: Add support for setting MSSQL Timezone in aws_db_instance ([#11247](https://github.com/hashicorp/terraform/issues/11247))
 * provider/aws: CloudFormation YAML template support ([#11121](https://github.com/hashicorp/terraform/issues/11121))
 * provider/aws: Remove hardcoded https from the ecr repository ([#11307](https://github.com/hashicorp/terraform/issues/11307))
 * provider/aws: Implement CloudFront Lambda Function Associations ([#11291](https://github.com/hashicorp/terraform/issues/11291))
 * provider/aws: Remove MaxFrameRate default on ElasticTranscoderPreset ([#11340](https://github.com/hashicorp/terraform/issues/11340))
 * provider/aws: Allow ARN Identifier to be set for different partitions ([#11359](https://github.com/hashicorp/terraform/issues/11359))
 * provider/aws: Allow bypassing region validation ([#11358](https://github.com/hashicorp/terraform/issues/11358))
 * provider/aws: Added a s3_bucket domain name attribute ([#10088](https://github.com/hashicorp/terraform/issues/10088))
 * provider/aws: Add DiffSupressFunction to aws_db_instance's engine_version ([#11369](https://github.com/hashicorp/terraform/issues/11369))
 * provider/archive: Adding support for multiple source contents ([#11271](https://github.com/hashicorp/terraform/issues/11271))
 * provider/azurerm: add caching support for virtual_machine data_disks ([#11142](https://github.com/hashicorp/terraform/issues/11142))
 * provider/azurerm: make lb sub resources idempotent ([#11128](https://github.com/hashicorp/terraform/issues/11128))
 * provider/cloudflare: Add verification for record types and content ([#11197](https://github.com/hashicorp/terraform/issues/11197))
 * provider/datadog: Add aggregator method to timeboard graph resource ([#11206](https://github.com/hashicorp/terraform/issues/11206))
 * provider/fastly Add request_condition to backend definition ([#11238](https://github.com/hashicorp/terraform/issues/11238))
 * provider/google: Add subnetwork_project field to enable cross-project networking in instance templates ([#11110](https://github.com/hashicorp/terraform/issues/11110))
 * provider/google: Add support for encrypting a disk ([#11167](https://github.com/hashicorp/terraform/issues/11167))
 * provider/google: Add support for session_affinity to google_compute_region_backend_service ([#11228](https://github.com/hashicorp/terraform/issues/11228))
 * provider/google: Allow additional zones to be configured in GKE ([#11018](https://github.com/hashicorp/terraform/issues/11018))
 * provider/ignition: Allow empty dropin and content for systemd_units ([#11327](https://github.com/hashicorp/terraform/issues/11327))
 * provider/openstack: LoadBalancer Security Groups ([#11074](https://github.com/hashicorp/terraform/issues/11074))
 * provider/openstack: Volume Attachment Updates ([#11285](https://github.com/hashicorp/terraform/issues/11285))
 * provider/scaleway improve bootscript data source ([#11183](https://github.com/hashicorp/terraform/issues/11183))
 * provider/statuscake: Add support for StatusCake confirmation servers ([#11179](https://github.com/hashicorp/terraform/issues/11179))
 * provider/statuscake: Add support for Updating StatusCake contact_ids ([#7115](https://github.com/hashicorp/terraform/issues/7115))
 * provisioner/chef: Add support for named run-lists when using policyfiles ([#11215](https://github.com/hashicorp/terraform/issues/11215))
 * core: Add basic HTTP Auth for remote state backend ([#11301](https://github.com/hashicorp/terraform/issues/11301))

BUG FIXES:

 * command/fmt: Multiple `#` comments won't be separated by newlines. ([#11209](https://github.com/hashicorp/terraform/issues/11209))
 * command/fmt: Lists with a heredoc element that starts on the same line as the opening brace is formatted properly. ([#11208](https://github.com/hashicorp/terraform/issues/11208))
 * command/import: Provider configuration inheritance into modules works properly ([#11393](https://github.com/hashicorp/terraform/issues/11393))
 * command/import: Update help text to note that `-var` and `-var-file` work
 * provider/aws: Fix panic when querying VPC's main route table via data source ([#11134](https://github.com/hashicorp/terraform/issues/11134))
 * provider/aws: Allow creating aws_codecommit repository outside of us-east-1 ([#11177](https://github.com/hashicorp/terraform/issues/11177))
 * provider/aws: Fix issue destroying or updating CloudFront due to missing Lambda Function Associations parameters ([#11291](https://github.com/hashicorp/terraform/issues/11291))
 * provider/aws: Correct error messages are now returned if an `aws_autoscaling_lifecycle_hook` fails during creation ([#11360](https://github.com/hashicorp/terraform/issues/11360))
 * provider/aws: Fix issue updating/destroying Spot Fleet requests when using `terminate_instances_with_expiration` ([#10953](https://github.com/hashicorp/terraform/issues/10953))
 * provider/azurerm: use configured environment for storage clients ([#11159](https://github.com/hashicorp/terraform/issues/11159))
 * provider/google: removes region param from google_compute_backend_service ([#10903](https://github.com/hashicorp/terraform/issues/10903))
 * provider/ignition: allowing empty systemd.content when a dropin is provided ([#11216](https://github.com/hashicorp/terraform/issues/11216))
 * provider/openstack: Increase deletion timeout for router interfaces ([#11250](https://github.com/hashicorp/terraform/issues/11250))
 * provider/openstack: Fix Instance Metadata Deletion ([#11252](https://github.com/hashicorp/terraform/issues/11252))
 * provider/scaleway: Rename Scaleway provider parameters to match more closely to the API ([#10874](https://github.com/hashicorp/terraform/issues/10874))
 * provider/vault: Remove user input for optional vault provider fields ([#11082](https://github.com/hashicorp/terraform/issues/11082))
 * provider/vsphere: Set deviceID to 0 if one 1 network interface in vsphere_virtual_machine ([#8276](https://github.com/hashicorp/terraform/issues/8276))
 * provisioner/remote-exec: fail on first inline script with bad exit code ([#11155](https://github.com/hashicorp/terraform/issues/11155))

## 0.8.4 (January 11, 2017)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * We have removed the `Arukas` provider that was added in v0.8.3 for this release. Unfortunately we found the
 new provider included a dependency that would not compile and run on Windows operating systems. For now the
 provider has been removed and we hope to work to reintroduce it for all platforms in the near future. Going forward we will also be taking additional steps in our build testing to ensure Terraform builds on all platforms before release.

## 0.8.3 (January 10, 2017)

FEATURES:

 * **New Provider:**  `Arukas` ([#10862](https://github.com/hashicorp/terraform/issues/10862))
 * **New Provider:**  `Ignition` ([#6189](https://github.com/hashicorp/terraform/issues/6189))
 * **New Provider:**  `OpsGenie` ([#11012](https://github.com/hashicorp/terraform/issues/11012))
 * **New Data Source:**  `aws_vpc_peering_connection` ([#10913](https://github.com/hashicorp/terraform/issues/10913))
 * **New Resource:**  `aws_codedeploy_deployment_config` ([#11062](https://github.com/hashicorp/terraform/issues/11062))
 * **New Resource:**  `azurerm_container_registry` ([#10973](https://github.com/hashicorp/terraform/issues/10973))
 * **New Resource:**  `azurerm_eventhub_authorization_rule` ([#10971](https://github.com/hashicorp/terraform/issues/10971))
 * **New Resource:**  `azurerm_eventhub_consumer_group` ([#9902](https://github.com/hashicorp/terraform/issues/9902))

IMPROVEMENTS:

 * command/fmt: Show filename on parse error ([#10923](https://github.com/hashicorp/terraform/issues/10923))
 * provider/archive: `archive_file` now exports `output_md5` attribute in addition to existing SHA1 and Base64 SHA256 hashes. ([#10851](https://github.com/hashicorp/terraform/issues/10851))
 * provider/aws: Add `most_recent` to the `ebs_snapshot` data source ([#10986](https://github.com/hashicorp/terraform/issues/10986))
 * provider/aws: Add support for instance tenancy in `aws_opsworks_instance` ([#10885](https://github.com/hashicorp/terraform/issues/10885))
 * provider/aws: Added a validation for security group rule types ([#10864](https://github.com/hashicorp/terraform/issues/10864))
 * provider:aws: Add support for updating aws_emr_cluster parameters ([#11008](https://github.com/hashicorp/terraform/issues/11008))
 * provider/aws: Add Placement Constraints to `aws_ecs_task_definition` ([#11030](https://github.com/hashicorp/terraform/issues/11030))
 * provider/aws: Increasing timeout for redshift cluster creation to 75 minutes ([#11041](https://github.com/hashicorp/terraform/issues/11041))
 * provider/aws: Add support for content_handling to aws_api_gateway_integration_response ([#11002](https://github.com/hashicorp/terraform/issues/11002))
 * provider/aws: Add S3 bucket name validation ([#11116](https://github.com/hashicorp/terraform/issues/11116))
 * provider/aws: Add Route53 Record type validation ([#11119](https://github.com/hashicorp/terraform/issues/11119))
 * provider/azurerm: support non public clouds ([#11026](https://github.com/hashicorp/terraform/issues/11026))
 * provider/azurerm: Azure resource providers which are already registered are no longer re-registered. ([#10991](https://github.com/hashicorp/terraform/issues/10991))
 * provider/docker: Add network create --internal flag support ([#10932](https://github.com/hashicorp/terraform/issues/10932))
 * provider/docker: Add support for a list of pull_triggers within the docker_image resource. ([#10845](https://github.com/hashicorp/terraform/issues/10845))
 * provider/pagerduty Add delete support to `pagerduty_service_integration` ([#10891](https://github.com/hashicorp/terraform/issues/10891))
 * provider/postgresql Add permissions support to `postgresql_schema` as nested `policy` attributes ([#10808](https://github.com/hashicorp/terraform/issues/10808))

BUG FIXES:

 * core: Properly expand sets as lists from a flatmap ([#11042](https://github.com/hashicorp/terraform/issues/11042))
 * core: Disallow root modules named "root" as a temporary workaround ([#11099](https://github.com/hashicorp/terraform/issues/11099))
 * command/fmt: Lists of heredocs format properly ([#10947](https://github.com/hashicorp/terraform/issues/10947))
 * command/graph: Fix crash when `-type=legacy` ([#11095](https://github.com/hashicorp/terraform/issues/11095))
 * provider/aws: Guard against nil change output in `route53_zone` that causes panic ([#10798](https://github.com/hashicorp/terraform/issues/10798))
 * provider/aws: Reworked validateArn function to handle empty values ([#10833](https://github.com/hashicorp/terraform/issues/10833))
 * provider/aws: Set `aws_autoscaling_policy` `metric_aggregation_type` to be Computed ([#10904](https://github.com/hashicorp/terraform/issues/10904))
 * provider/aws: `storage_class` is now correctly treated as optional when configuring replication for `aws_s3_bucket` resources. ([#10921](https://github.com/hashicorp/terraform/issues/10921))
 * provider/aws: `user_data` on `aws_launch_configuration` resources is only base 64 encoded if the value provided is not already base 64 encoded. ([#10871](https://github.com/hashicorp/terraform/issues/10871))
 * provider/aws: Add snapshotting to the list of pending state for elasticache ([#10965](https://github.com/hashicorp/terraform/issues/10965))
 * provider/aws: Add support for updating tags in aws_emr_cluster ([#11003](https://github.com/hashicorp/terraform/issues/11003))
 * provider/aws: Fix the normalization of AWS policy statements ([#11009](https://github.com/hashicorp/terraform/issues/11009))
 * provider/aws: data_source_aws_iam_server_certificate latest should be bool not string causes panic ([#11016](https://github.com/hashicorp/terraform/issues/11016))
 * provider/aws: Fix typo in aws_redshift_cluster causing security groups to not allow update ([#11025](https://github.com/hashicorp/terraform/issues/11025))
 * provider/aws: Set `key_name` in `aws_key_pair` if omited in configuration ([#10987](https://github.com/hashicorp/terraform/issues/10987))
 * provider/aws: Updating the aws_efs_mount_target dns_name ([#11023](https://github.com/hashicorp/terraform/issues/11023))
 * provider/aws: Validate window time format for snapshot times and backup windows on RDS and ElastiCache resources ([#11089](https://github.com/hashicorp/terraform/issues/11089))
 * provider/aws: aws_db_instance restored from snapshot had problem with subnet_group ([#11050](https://github.com/hashicorp/terraform/issues/11050))
 * provider/aws: Allow disabled access_log in ELB ([#11120](https://github.com/hashicorp/terraform/issues/11120))
 * provider/azurerm: fix update protocol for lb_probe ([#11125](https://github.com/hashicorp/terraform/issues/11125))
 * provider/google: Fix backwards incompatibility around create_timeout in instances ([#10858](https://github.com/hashicorp/terraform/issues/10858))
 * provider/google: google_compute_instance_group_manager update_strategy not properly read ([#10174](https://github.com/hashicorp/terraform/issues/10174))
 * provider/openstack: Handle `PENDING_UPDATE` status with LBaaS v2 members ([#10875](https://github.com/hashicorp/terraform/issues/10875))
 * provider/rancher: Add 'finishing-upgrade' state to rancher stack ([#11019](https://github.com/hashicorp/terraform/issues/11019))


## 0.8.2 (December 21, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * `aws_lambda_function` Please note that `runtime` is now a required field as AWS have deprecated the use of nodejs 0.10 in lambda functions ([#9724](https://github.com/hashicorp/terraform/issues/9724))

FEATURES:

 * **New Provider:**  `New Relic` ([#10317](https://github.com/hashicorp/terraform/issues/10317))
 * **New Resource:**  `aws_ses_configuration_set` ([#10735](https://github.com/hashicorp/terraform/issues/10735))
 * **New Resource:**  `aws_ses_event_destination` ([#10735](https://github.com/hashicorp/terraform/issues/10735))
 * **New Resource:**  `azurerm_redis_cache` ([#10184](https://github.com/hashicorp/terraform/issues/10184))
 * **New Resource:**  `ultradns_dirpool` ([#9788](https://github.com/hashicorp/terraform/issues/9788))
 * **New Resource:**  `ultradns_probe_http` ([#9788](https://github.com/hashicorp/terraform/issues/9788))
 * **New Resource:**  `ultradns_probe_ping` ([#9788](https://github.com/hashicorp/terraform/issues/9788))
 * **New Resource:**  `ultradns_record` ([#9788](https://github.com/hashicorp/terraform/issues/9788))
 * **New Resource:**  `ultradns_tcpool` ([#9788](https://github.com/hashicorp/terraform/issues/9788))
 * **New Data Source:**  `aws_iam_account_alias` ([#10804](https://github.com/hashicorp/terraform/issues/10804))

IMPROVEMENTS:

 * provider/aws: Add support for BinaryMediaTypes and ContentHandling to AWS API Gateway ([#10776](https://github.com/hashicorp/terraform/issues/10776))
 * provider/aws: Deprecated aws_lambda_function nodejs runtime in favor of nodejs4.3 ([#9724](https://github.com/hashicorp/terraform/issues/9724))
 * provider/aws: Support updating of aws_db_instance db_subnet_group_name ([#10818](https://github.com/hashicorp/terraform/issues/10818))
 * provider/aws: Allow update to RDS password when restoring from snapshot ([#8622](https://github.com/hashicorp/terraform/issues/8622))
 * provider/azurerm: add support for tags to dns_zone ([#10750](https://github.com/hashicorp/terraform/issues/10750))
 * provider/pagerduty pagerduty_schedule - support for start_day_of_week (schedule restriction) ([#10069](https://github.com/hashicorp/terraform/issues/10069))
 * state/remote/swift: add support for token authentication ([#10866](https://github.com/hashicorp/terraform/issues/10866))

BUG FIXES:

 * core: Improve validation for provider aliases to allow inheritance in moduels. ([#10807](https://github.com/hashicorp/terraform/issues/10807))
 * core: Math operations always prefer floating point if an argument is floating point. ([#10886](https://github.com/hashicorp/terraform/issues/10886))
 * core: Strings are implicitly converted to integers/floats for comparison. ([#10886](https://github.com/hashicorp/terraform/issues/10886))
 * provider/aws: Fixed crash in `data_source_ami` with empty `owner` value ([#10763](https://github.com/hashicorp/terraform/issues/10763))
 * provider/aws: Require `master_username` and `master_password` if no snapshot given in Redshift Cluster ([#9837](https://github.com/hashicorp/terraform/issues/9837))
 * provider/azurerm: fix network_interface.ip_configuration hash for load balancers ([#10834](https://github.com/hashicorp/terraform/issues/10834))
 * provider/docker: Fix regression, 'cert_path' stop working ([#10801](https://github.com/hashicorp/terraform/issues/10801))
 * provider/google: Use node_version during google_container_cluster creation ([#10817](https://github.com/hashicorp/terraform/issues/10817))
 * provider/openstack: Handle Volume Creation Errors ([#10821](https://github.com/hashicorp/terraform/issues/10821))

## 0.8.1 (December 14, 2016)

IMPROVEMENTS:

 * provider/aws: Support eu-west-2 ([#10470](https://github.com/hashicorp/terraform/issues/10470))
 * provider/aws: Improved the SNS topic subscription protocols validation ([#10704](https://github.com/hashicorp/terraform/issues/10704))
 * providers/google: Add subnetwork_project field to enable cross-project networking ([#9662](https://github.com/hashicorp/terraform/issues/9662))
 * provider/pagerduty: Allow 'team_responder' role for pagerduty_user resource ([#10728](https://github.com/hashicorp/terraform/issues/10728))

BUG FIXES:

  * core: Handle whitespace around the key in the `-var` flag. ([#10717](https://github.com/hashicorp/terraform/issues/10717))
  * core: `terraform` block works in the presence of `_override` files ([#10715](https://github.com/hashicorp/terraform/issues/10715))
  * core: Fix error when a provider in a module only referenced a variable ([#10719](https://github.com/hashicorp/terraform/issues/10719))
  * core: Destroy ordering for resources that depend on each other across modules is correct ([#745](https://github.com/hashicorp/terraform/issues/745))

DEPRECATION REMOVALS:

  * provider/aws: Removed deprecated `parameter_group` from `aws_rds_cluster` ([#10733](https://github.com/hashicorp/terraform/issues/10733))

## 0.8.0 (December 13, 2016)

**This is the complete 0.7.13 to 0.8 CHANGELOG. Below this section we
also have a 0.8.0-rc3 to 0.8.0 final CHANGELOG.**

BACKWARDS INCOMPATIBILITIES / NOTES:

 * `template_file` _inline_ templates must escape their variable usage. What
   was previously `${foo}` must now be `$${foo}`. Note that this is only
   for _inline_ templates. Templates read from files are unchanged. ([#9698](https://github.com/hashicorp/terraform/issues/9698))
 * Escape sequences used to require double-escaping when used within interpolations.
   You now must only escape once (which is the expected/typical behavior).
   For example: `${replace(var.foo, "\\", "\\\\")}` is correct. Before,
   that would cause very strange behavior. However, this may break existing
   configurations which found a level of escape sequences to work. Check
   `terraform plan` for incorrect output.
 * Math operators now follow the standard order of operations: *, /, % followed
   by +, -. See the updated interpolation docs for more information. You can
   continue to force ordering with parentheses.
 * Strings in configuration can no longer contain unescaped newlines. For
   unescaped newlines, heredocs must be used

 * provider/aws: Anywhere where we can specify kms_key_id must now be a valid KMS Key ID ARN to stop continual diffs
 * provider/chef: The chef provider now accepts `key_material` as an alternative to
   `private_key_pem`. The `private_key_pem` attribute will be deprecated in a
    future release
 * provider/postgres: `ssl_mode` has been renamed `sslmode` to match common usage ([#10682](https://github.com/hashicorp/terraform/issues/10682))

DEPRECATION REMOVALS:

 * The `template_file` resource no longer accepts a direct file path for the
   `template` attribute. You may either specify a path wrapped in a `file`
   function or specify a file path with the `filepath` attribute. This was
   deprecated during 0.7.x.

FEATURES:

 * **New command:** `terraform console`, an interactive console for experimenting
   with and using interpolations. ([#10093](https://github.com/hashicorp/terraform/issues/10093))
 * **Terraform version requirement in configuration.** You can now specify
   a Terraform version requirement in configuration and modules. ([#10080](https://github.com/hashicorp/terraform/issues/10080))
 * **Conditional values:** You can now use conditionals to determine the values
   of attributes. For example: `count = "${var.env == "prod" ? 1 : 0}"`.
 * **`depends_on` can reference modules.** This allows a resource or output
   to depend on everything within a module. ([#10076](https://github.com/hashicorp/terraform/issues/10076))
 * **`output` supports `depends_on`.** This is useful when the output depends
   on a certain ordering to happen that can't be represented with interpolations.
   ([#10072](https://github.com/hashicorp/terraform/issues/10072))
 * Providers and resources are now notified by Terraform core to "stop" when
   an interrupt is received, allowing resources to gracefully exit much, much
   faster. ([#9607](https://github.com/hashicorp/terraform/issues/9607))
 * The `import` command can now specify a provider alias to use. ([#10310](https://github.com/hashicorp/terraform/issues/10310))
 * The `import` command will now read provider configuration from Terraform
   configuration files (including loading tfvars files and so on).
   ([#9809](https://github.com/hashicorp/terraform/issues/9809))

 * **New Provider:** `external` ([#8768](https://github.com/hashicorp/terraform/issues/8768))
 * **New Provider:** `nomad` ([#9538](https://github.com/hashicorp/terraform/issues/9538))
 * **New Provider:** `rancher` ([#9173](https://github.com/hashicorp/terraform/issues/9173))
 * **New Provider:** `vault` ([#9158](https://github.com/hashicorp/terraform/issues/9158))
 * **New Provider:** `Icinga2` ([#8306](https://github.com/hashicorp/terraform/issues/8306))
 * **New Resource:** `aws_ebs_snapshot` ([#10017](https://github.com/hashicorp/terraform/issues/10017))
 * **New Resource:** `aws_lightsail_domain` ([#10637](https://github.com/hashicorp/terraform/issues/10637))
 * **New Resource:** `aws_lightsail_key_pair` ([#10583](https://github.com/hashicorp/terraform/issues/10583))
 * **New Resource:** `aws_lightsail_instance` ([#10473](https://github.com/hashicorp/terraform/issues/10473))
 * **New Resource:** `aws_opsworks_rds_db_instance` ([#10294](https://github.com/hashicorp/terraform/issues/10294))
 * **New Resource:** `aws_snapshot_create_volume_permission` ([#9891](https://github.com/hashicorp/terraform/issues/9891))
 * **New Resource:** `aws_vpc_endpoint_route_table_association` ([#10137](https://github.com/hashicorp/terraform/issues/10137))
 * **New Resource:** `google_compute_health_check` ([#10453](https://github.com/hashicorp/terraform/issues/10453))
 * **New Resource:** `google_compute_region_backend_service` ([#10453](https://github.com/hashicorp/terraform/issues/10453))
 * **New Resource:** `openstack_blockstorage_volume_attach_v2` ([#10259](https://github.com/hashicorp/terraform/issues/10259))
 * **New Resource:** `openstack_compute_volume_attach_v2` ([#10260](https://github.com/hashicorp/terraform/issues/10260))
 * **New Data Source:** `aws_ebs_snapshot` ([#10017](https://github.com/hashicorp/terraform/issues/10017))
 * **New Data Source:** `aws_eip` ([#9833](https://github.com/hashicorp/terraform/issues/9833))
 * **New Data Source:** `aws_iam_server_certificate` ([#10558](https://github.com/hashicorp/terraform/issues/10558))
 * **New Data Source:** `aws_route_table` ([#10301](https://github.com/hashicorp/terraform/issues/10301))
 * **New Data Source:** `aws_route53_zone` ([#9766](https://github.com/hashicorp/terraform/issues/9766))
 * **New Data Source:** `aws_vpc_endpoint_services` ([#10261](https://github.com/hashicorp/terraform/issues/10261))
 * **New Data Source:** `pagerduty_user` ([#10541](https://github.com/hashicorp/terraform/issues/10541))
 * **New Interpolation Function:** `timestamp` ([#10475](https://github.com/hashicorp/terraform/issues/10475))
 * core: allow outputs to have descriptions ([#9722](https://github.com/hashicorp/terraform/issues/9722))
 * state/azure: support passing of lease ID when writing storage blob ([#10115](https://github.com/hashicorp/terraform/issues/10115))

IMPROVEMENTS:

 * core: Human-friendly error when a computed count is used. ([#10060](https://github.com/hashicorp/terraform/issues/10060))
 * core: Maps across multiple input sources (files, CLI, env vars) are merged. ([#10654](https://github.com/hashicorp/terraform/issues/10654))
 * core: SIGTERM also triggers graceful shutdown in addition to SIGINT ([#10534](https://github.com/hashicorp/terraform/issues/10534))
 * core: Plan will show deposed-only destroys for create-before-destroy resources. ([#10404](https://github.com/hashicorp/terraform/issues/10404))
 * command/plan: Show warning when a plan file is given as input to make behavior clear. ([#10639](https://github.com/hashicorp/terraform/issues/10639))
 * helper/schema: only map, list, and set elements that are actually causing
     a resource to destroy/create are marked as "requires new". ([#9613](https://github.com/hashicorp/terraform/issues/9613))
 * provider/aws: Add support for AWS CA Central 1 Region ([#10618](https://github.com/hashicorp/terraform/issues/10618))
 * provider/aws: Allow importing of aws_iam_role, aws_iam_role_policy and aws_iam_policy ([#9398](https://github.com/hashicorp/terraform/issues/9398))
 * provider/aws: Added s3 bucket region attribute management ([#10482](https://github.com/hashicorp/terraform/issues/10482))
 * provider/aws: Added SQS FIFO queues ([#10614](https://github.com/hashicorp/terraform/issues/10614))
 * provider/aws: Addition of suspended_processes to aws_autoscaling_group ([#10096](https://github.com/hashicorp/terraform/issues/10096))
 * provider/aws: added auto_minor_version_upgrade on aws_rds_cluster_insstance ([#10284](https://github.com/hashicorp/terraform/issues/10284))
 * provider/aws: Add JSON validation to the aws_iam_policy resource ([#10239](https://github.com/hashicorp/terraform/issues/10239))
 * provider/aws: Support MFA delete for s3 bucket versioning ([#10020](https://github.com/hashicorp/terraform/issues/10020))
 * provider/aws: Enable DeleteOnTermination in ENI when created by spot fleet ([#9922](https://github.com/hashicorp/terraform/issues/9922))
 * provider/aws: Enforced kms_key_* attributes to be ARNs ([#10356](https://github.com/hashicorp/terraform/issues/10356))
 * provider/aws: IPv6 Support To Cloudfront ([#10332](https://github.com/hashicorp/terraform/issues/10332))
 * provider/aws: Support import of aws_iam_instance_profile ([#10436](https://github.com/hashicorp/terraform/issues/10436))
 * provider/aws: Increase `aws_emr_cluster` timeout ([#10444](https://github.com/hashicorp/terraform/issues/10444))
 * provider/aws: Support Automatic Rollback of CodeDeploy deployments and CloudWatch Alarms for a Deployment Group ([#9039](https://github.com/hashicorp/terraform/issues/9039))
 * provider/aws: Add support for termination protection and autotermination to EMR ([#10252](https://github.com/hashicorp/terraform/issues/10252))
 * provider/aws: Add "no_device" support to ephemeral block devices ([#10547](https://github.com/hashicorp/terraform/issues/10547))
 * provider/aws: Added S3 Bucket replication ([#10552](https://github.com/hashicorp/terraform/issues/10552))
 * provider/aws: Add `pgp_key` to `aws_iam_access_key` to protect key. ([#10615](https://github.com/hashicorp/terraform/issues/10615))
 * provider/azurerm: make DiskSizeGB optional for azurerm_virtual_machine data_disks ([#10232](https://github.com/hashicorp/terraform/issues/10232))
 * provider/azurerm support `license_type` virtual_machine property ([#10539](https://github.com/hashicorp/terraform/issues/10539))
 * provider/azurerm: support import of routes, fix route_table ([#10389](https://github.com/hashicorp/terraform/issues/10389))
 * provider/azurerm: enable import of more resources ([#10195](https://github.com/hashicorp/terraform/issues/10195))
 * provider/azurerm: create common schema for location field, add diff suppress ([#10409](https://github.com/hashicorp/terraform/issues/10409))
 * provider/chef: Migrate Chef to use KEY_MATERIAL rather than using a Pem file ([#10105](https://github.com/hashicorp/terraform/issues/10105))
 * provider/cloudstack: Add option to set a custom `network_domain` for `cloudstack_network` ([#10638](https://github.com/hashicorp/terraform/issues/10638))
 * provider/cloudstack: Support using secondary IP addresses with the `cloudstack_static_nat` resource ([#10420](https://github.com/hashicorp/terraform/issues/10420))
 * provider/cloudstack: Support using secondary IP addresses with the `cloudstack_port_forward` resource ([#10638](https://github.com/hashicorp/terraform/issues/10638))
 * provider/datadog: Make monitor thresholds optional. ([#10526](https://github.com/hashicorp/terraform/issues/10526))
 * provider/datadog: Improve datadog timeboard support ([#10027](https://github.com/hashicorp/terraform/issues/10027))
 * provider/docker: Upload files into container before first start ([#9520](https://github.com/hashicorp/terraform/issues/9520))
 * provider/docker: authentication via values instead of files ([#10151](https://github.com/hashicorp/terraform/issues/10151))
 * provider/fastly add origin shielding ([#10677](https://github.com/hashicorp/terraform/issues/10677))
 * provider/fastly: add ssl_hostname option ([#9629](https://github.com/hashicorp/terraform/issues/9629))
 * provider/github: supports importing resources ([#10382](https://github.com/hashicorp/terraform/issues/10382))
 * provider/google: Add support for Internal Load Balancing ([#10453](https://github.com/hashicorp/terraform/issues/10453))
 * provider/google: Add Service Accounts resource ([#9946](https://github.com/hashicorp/terraform/issues/9946))
 * provider/google: Instances and templates now both support `metadata_startup_script` and `metadata.startup-script`. ([#10537](https://github.com/hashicorp/terraform/issues/10537))
 * provider/google: Added support for session affinity to compute_backend_service ([#10387](https://github.com/hashicorp/terraform/issues/10387))
 * provider/google: Projects are now importable ([#10469](https://github.com/hashicorp/terraform/issues/10469))
 * provider/google: SSL certificates can now specify prefix instead of a full name ([#10684](https://github.com/hashicorp/terraform/issues/10684))
 * provider/openstack: Add Swauth/Swift Authentication ([#9943](https://github.com/hashicorp/terraform/issues/9943))
 * provider/openstack: Detect Region for Importing Resources ([#10509](https://github.com/hashicorp/terraform/issues/10509))
 * provider/postgresql: Improved support for many PostgreSQL resources ([#10682](https://github.com/hashicorp/terraform/issues/10682))
 * provider/postgresql: Added 'connect_timeout' argument to provider 'postgresql' ([#10380](https://github.com/hashicorp/terraform/issues/10380))
 * provider/rundeck: enable validation for multiple values in an array ([#8913](https://github.com/hashicorp/terraform/issues/8913))
 * provider/rundeck: Add support for scheduler to rundeck_job ([#9449](https://github.com/hashicorp/terraform/issues/9449))
 * state/remote/swift: Add support for versioning state file in swift and expiring versioned state ([#10055](https://github.com/hashicorp/terraform/issues/10055))

BUG FIXES:

 * core: Escape sequences in interpolations work in every case. ([#8709](https://github.com/hashicorp/terraform/issues/8709))
 * core: Maps in outputs with computed values are no longer removed. ([#9549](https://github.com/hashicorp/terraform/issues/9549))
 * core: Direct indexing into a computed list no longer errors. ([#10657](https://github.com/hashicorp/terraform/issues/10657))
 * core: Validate fails on invalid keys in `variable` blocks. ([#10658](https://github.com/hashicorp/terraform/issues/10658))
 * core: Validate that only a single `lifecycle` block exists per rource. ([#10656](https://github.com/hashicorp/terraform/issues/10656))
 * core: When destroying, the resources of a provider that depends on another resource are destroyed first. ([#10659](https://github.com/hashicorp/terraform/issues/10659))
 * core: Catch parse errors for null characters mid-file ([#9134](https://github.com/hashicorp/terraform/issues/9134))
 * core: Remove extra dot from state command backup files ([#10300](https://github.com/hashicorp/terraform/issues/10300))
 * core: Validate data sources do not have provisioners ([#10318](https://github.com/hashicorp/terraform/issues/10318))
 * core: Disable checkpoint settings take effect ([#10206](https://github.com/hashicorp/terraform/issues/10206))
 * core: Changed attribute console output shows up on Windows. ([#10417](https://github.com/hashicorp/terraform/issues/10417))
 * core: Destroying deposed resources in create before destroy waits until the creation step of its specific index. (0.8 regression) ([#10416](https://github.com/hashicorp/terraform/issues/10416))
 * core: Certain invalid configurations will no longer print "illegal". ([#10448](https://github.com/hashicorp/terraform/issues/10448))
 * core: Fix a crash that could occur when multiple deposed instances exist. ([#10504](https://github.com/hashicorp/terraform/issues/10504))
 * core: Fix a diff mismatch error that could happen when a resource depends on a count resource being decreased. ([#10522](https://github.com/hashicorp/terraform/issues/10522))
 * core: On Unix machines if `getent` is not available, fall back to shell to find home dir. ([#10515](https://github.com/hashicorp/terraform/issues/10515))
 * command/fmt: Multiline comments aren't indented every fmt. ([#6524](https://github.com/hashicorp/terraform/issues/6524))
 * communicator/ssh: Avoid race that could cause parallel remote execs on the same host to overwrite each other ([#10549](https://github.com/hashicorp/terraform/issues/10549))
 * provider/aws: Added Lambda function guard when needed attributes are not set ([#10663](https://github.com/hashicorp/terraform/issues/10663))
 * provider/aws: Allow import of aws_security_groups with more than one source_security_group_id rule ([#9477](https://github.com/hashicorp/terraform/issues/9477))
 * provider/aws: Allow setting the DB Instance name when restoring from a snapshot ([#10664](https://github.com/hashicorp/terraform/issues/10664))
 * provider/aws: Fix issue importing `aws_vpc_peering_connection` ([#10635](https://github.com/hashicorp/terraform/issues/10635))
 * provider/aws: Fixed deletion of aws_api_gateway_base_path_mapping with empty path ([#10177](https://github.com/hashicorp/terraform/issues/10177))
 * provider/aws: Fix issue removing Lambda environment variables ([#10492](https://github.com/hashicorp/terraform/issues/10492))
 * provider/aws: Skip VPC endpoint routes when removing default route table's routes ([#10303](https://github.com/hashicorp/terraform/issues/10303))
 * provider/aws: Do not return a root device for instance store backed AMIs. ([#9483](https://github.com/hashicorp/terraform/issues/9483))
 * provider/aws: resource_aws_opsworks_application does not accept document_root parameter ([#10477](https://github.com/hashicorp/terraform/issues/10477))
 * provider/aws: bug fix when specifying level on aws_opsworks_permission ([#10394](https://github.com/hashicorp/terraform/issues/10394))
 * provider/aws: cloudfront distribution 404 should mark as gone ([#10281](https://github.com/hashicorp/terraform/issues/10281))
 * provider/aws: Assign correct number of core instances (n-1) to aws-emr-cluster on update ([#10529](https://github.com/hashicorp/terraform/issues/10529))
 * provider/aws: Allow update of Service role on a CodeDeploy deployment group ([#9866](https://github.com/hashicorp/terraform/issues/9866))
 * provider/aws: fixed the api_gw_domain_name replace operation ([#10179](https://github.com/hashicorp/terraform/issues/10179))
 * provider/aws: Forces the API GW domain name certificates to recreate the resource ([#10588](https://github.com/hashicorp/terraform/issues/10588))
 * provider/aws: Validate `effect` in aws_iam_policy_document data source ([#10021](https://github.com/hashicorp/terraform/issues/10021))
 * provider/azurerm: fix virtual_machine reading plan as the wrong type ([#10626](https://github.com/hashicorp/terraform/issues/10626))
 * provider/azurerm: Prevent null reference when reading boot_diagnostics settings in azurerm_virtual_machine ([#10283](https://github.com/hashicorp/terraform/issues/10283))
 * provider/azurerm: azurerm_availability_set not is ForceNew for UpdateDomain and FaultDomain ([#10545](https://github.com/hashicorp/terraform/issues/10545))
 * provider/azurerm: fix servicebus_topic max_size_in_megabytes for premium namespaces ([#10611](https://github.com/hashicorp/terraform/issues/10611))
 * provider/azurerm: set ForceNew for storage image and OS disk of virtual_machine ([#10340](https://github.com/hashicorp/terraform/issues/10340))
 * provider/datadog: Refactor monitor tags to a list instead of a map. ([#10570](https://github.com/hashicorp/terraform/issues/10570))
 * provider/datadog 9869: Validate credentials when initialising client. ([#10567](https://github.com/hashicorp/terraform/issues/10567))
 * provider/openstack: More Import and Region Fixes ([#10662](https://github.com/hashicorp/terraform/issues/10662))
 * provider/openstack: Fix Ordering of Port Allowed Address Pairs ([#10250](https://github.com/hashicorp/terraform/issues/10250))
 * provider/template: No file path error when setting template to `/` ([#10297](https://github.com/hashicorp/terraform/issues/10297))

## 0.8.0 from 0.8.0-rc3 (December 13, 2016)

**This only includes changes from 0.8.0-rc3 to 0.8.0 final. The section above
has the complete 0.7.x to 0.8.0 CHANGELOG.**

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/postgres: `ssl_mode` has been renamed `sslmode` to match common usage ([#10682](https://github.com/hashicorp/terraform/issues/10682))

FEATURES:

 * **New Provider:** `Icinga2` ([#8306](https://github.com/hashicorp/terraform/issues/8306))
 * **New Resource:** `aws_lightsail_domain` ([#10637](https://github.com/hashicorp/terraform/issues/10637))
 * **New Resource:** `aws_lightsail_key_pair` ([#10583](https://github.com/hashicorp/terraform/issues/10583))
 * **New Resource:** `aws_snapshot_create_volume_permission` ([#9891](https://github.com/hashicorp/terraform/issues/9891))
 * **New Resource:** `google_compute_health_check` ([#10453](https://github.com/hashicorp/terraform/issues/10453))
 * **New Resource:** `google_compute_region_backend_service` ([#10453](https://github.com/hashicorp/terraform/issues/10453))
 * **New Data Source:**  `aws_eip` ([#9833](https://github.com/hashicorp/terraform/issues/9833))
 * **New Data Source:**  `aws_route53_zone` ([#9766](https://github.com/hashicorp/terraform/issues/9766))
 * **New Data Source:**  `aws_vpc_endpoint_services` ([#10261](https://github.com/hashicorp/terraform/issues/10261))

IMPROVEMENTS:

 * command/plan: Show warning when a plan file is given as input to make behavior clear. ([#10639](https://github.com/hashicorp/terraform/issues/10639))
 * core: Maps across multiple input sources (files, CLI, env vars) are merged. ([#10654](https://github.com/hashicorp/terraform/issues/10654))
 * provider/aws: Add support for AWS CA Central 1 Region ([#10618](https://github.com/hashicorp/terraform/issues/10618))
 * provider/aws: Added SQS FIFO queues ([#10614](https://github.com/hashicorp/terraform/issues/10614))
 * provider/aws: Support MFA delete for s3 bucket versioning ([#10020](https://github.com/hashicorp/terraform/issues/10020))
 * provider/aws: Enable DeleteOnTermination in ENI when created by spot fleet ([#9922](https://github.com/hashicorp/terraform/issues/9922))
 * provider/cloudstack: Add option to set a custom `network_domain` for `cloudstack_network` ([#10638](https://github.com/hashicorp/terraform/issues/10638))
 * provider/cloudstack: Support using secondary IP addresses with the `cloudstack_port_forward` resource ([#10638](https://github.com/hashicorp/terraform/issues/10638))
 * provider/fastly add origin shielding ([#10677](https://github.com/hashicorp/terraform/issues/10677))
 * provider/google: Add support for Internal Load Balancing ([#10453](https://github.com/hashicorp/terraform/issues/10453))
 * provider/google: SSL certificates can now specify prefix instead of a full name ([#10684](https://github.com/hashicorp/terraform/issues/10684))
 * provider/postgresql: Improved support for many PostgreSQL resources ([#10682](https://github.com/hashicorp/terraform/issues/10682))
 * provider/rundeck: enable validation for multiple values in an array ([#8913](https://github.com/hashicorp/terraform/issues/8913))
 * provider/rundeck: Add support for scheduler to rundeck_job ([#9449](https://github.com/hashicorp/terraform/issues/9449))

BUG FIXES:

 * core: Direct indexing into a computed list no longer errors. ([#10657](https://github.com/hashicorp/terraform/issues/10657))
 * core: Validate fails on invalid keys in `variable` blocks. ([#10658](https://github.com/hashicorp/terraform/issues/10658))
 * core: Validate that only a single `lifecycle` block exists per rource. ([#10656](https://github.com/hashicorp/terraform/issues/10656))
 * core: When destroying, the resources of a provider that depends on another resource are destroyed first. ([#10659](https://github.com/hashicorp/terraform/issues/10659))
 * provider/aws: Added Lambda function guard when needed attributes are not set ([#10663](https://github.com/hashicorp/terraform/issues/10663))
 * provider/aws: Allow import of aws_security_groups with more than one source_security_group_id rule ([#9477](https://github.com/hashicorp/terraform/issues/9477))
 * provider/aws: Allow setting the DB Instance name when restoring from a snapshot ([#10664](https://github.com/hashicorp/terraform/issues/10664))
 * provider/aws: Fix issue importing `aws_vpc_peering_connection` ([#10635](https://github.com/hashicorp/terraform/issues/10635))
 * provider/aws: Fixed deletion of aws_api_gateway_base_path_mapping with empty path ([#10177](https://github.com/hashicorp/terraform/issues/10177))
 * provider/aws: Fix issue removing Lambda environment variables ([#10492](https://github.com/hashicorp/terraform/issues/10492))
 * provider/azurerm: fix virtual_machine reading plan as the wrong type ([#10626](https://github.com/hashicorp/terraform/issues/10626))
 * provider/azurerm: set ForceNew for storage image and OS disk of virtual_machine ([#10340](https://github.com/hashicorp/terraform/issues/10340))
 * provider/openstack: More Import and Region Fixes ([#10662](https://github.com/hashicorp/terraform/issues/10662))

## 0.8.0-rc3 (December 8, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * Variable, resource, provider, and module names may no longer start with
   a number or hyphen. Please see the upgrade guide for more information.

FEATURES:

 * **New Provider:** `external` ([#8768](https://github.com/hashicorp/terraform/issues/8768))
 * **New Provider:** `Rancher` ([#9173](https://github.com/hashicorp/terraform/issues/9173))
 * **New Data Source:** `aws_iam_server_certificate` ([#10558](https://github.com/hashicorp/terraform/issues/10558))
 * **New Data Source:** `pagerduty_user` ([#10541](https://github.com/hashicorp/terraform/issues/10541))
 * **New Resource:** `aws_opsworks_rds_db_instance` ([#10294](https://github.com/hashicorp/terraform/issues/10294))
 * **New Resource:** `aws_vpc_endpoint_route_table_association` ([#10137](https://github.com/hashicorp/terraform/issues/10137))
  * **New Resource:**  `aws_lightsail_instance` ([#10473](https://github.com/hashicorp/terraform/issues/10473))
IMPROVEMENTS:

 * core: SIGTERM also triggers graceful shutdown in addition to SIGINT ([#10534](https://github.com/hashicorp/terraform/issues/10534))
 * provider/aws: Add support for termination protection and autotermination to EMR ([#10252](https://github.com/hashicorp/terraform/issues/10252))
 * provider/aws: Add "no_device" support to ephemeral block devices ([#10547](https://github.com/hashicorp/terraform/issues/10547))
 * provider/aws: Added S3 Bucket replication ([#10552](https://github.com/hashicorp/terraform/issues/10552))
 * provider/aws: Add `pgp_key` to `aws_iam_access_key` to protect key. ([#10615](https://github.com/hashicorp/terraform/issues/10615))
 * provider/azurerm: make DiskSizeGB optional for azurerm_virtual_machine data_disks ([#10232](https://github.com/hashicorp/terraform/issues/10232))
 * provider/azurerm support `license_type` virtual_machine property ([#10539](https://github.com/hashicorp/terraform/issues/10539))
 * provider/datadog: Make monitor thresholds optional. ([#10526](https://github.com/hashicorp/terraform/issues/10526))
 * provider/datadog: Improve datadog timeboard support ([#10027](https://github.com/hashicorp/terraform/issues/10027))
 * provider/docker: Upload files into container before first start ([#9520](https://github.com/hashicorp/terraform/issues/9520))
 * provider/fastly: add ssl_hostname option ([#9629](https://github.com/hashicorp/terraform/issues/9629))
 * provider/openstack: Detect Region for Importing Resources ([#10509](https://github.com/hashicorp/terraform/issues/10509))
 * provider/google: Instances and templates now both support `metadata_startup_script` and `metadata.startup-script`. ([#10537](https://github.com/hashicorp/terraform/issues/10537))

BUG FIXES:

  * core: Fix a diff mismatch error that could happen when a resource depends on a count resource being decreased. ([#10522](https://github.com/hashicorp/terraform/issues/10522))
  * core: On Unix machines if `getent` is not available, fall back to shell to find home dir. ([#10515](https://github.com/hashicorp/terraform/issues/10515))
  * communicator/ssh: Avoid race that could cause parallel remote execs on the same host to overwrite each other ([#10549](https://github.com/hashicorp/terraform/issues/10549))
  * provider/aws: cloudfront distribution 404 should mark as gone ([#10281](https://github.com/hashicorp/terraform/issues/10281))
  * provider/aws: Assign correct number of core instances (n-1) to aws-emr-cluster on update ([#10529](https://github.com/hashicorp/terraform/issues/10529))
  * provider/aws: Allow update of Service role on a CodeDeploy deployment group ([#9866](https://github.com/hashicorp/terraform/issues/9866))
  * provider/aws: fixed the api_gw_domain_name replace operation ([#10179](https://github.com/hashicorp/terraform/issues/10179))
  * provider/aws: Forces the API GW domain name certificates to recreate the resource ([#10588](https://github.com/hashicorp/terraform/issues/10588))
  * provider/aws: Validate `effect` in aws_iam_policy_document data source ([#10021](https://github.com/hashicorp/terraform/issues/10021))
  * provider/azurem: azurerm_availability_set not is ForceNew for UpdateDomain and FaultDomain ([#10545](https://github.com/hashicorp/terraform/issues/10545))
  * provider/azurerm: fix servicebus_topic max_size_in_megabytes for premium namespaces ([#10611](https://github.com/hashicorp/terraform/issues/10611))
  * provider/datadog: Refactor monitor tags to a list instead of a map. ([#10570](https://github.com/hashicorp/terraform/issues/10570))
  * provider/datadog 9869: Validate credentials when initialising client. ([#10567](https://github.com/hashicorp/terraform/issues/10567))
  * provider/openstack: Fix Ordering of Port Allowed Address Pairs ([#10250](https://github.com/hashicorp/terraform/issues/10250))

## 0.8.0-rc2 (December 2, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * Strings in configuration can no longer contain unescaped newlines. For unescaped newlines, heredocs must be used
 * provider/aws: Anywhere where we can specify kms_key_id must now be a valid KMS Key ID ARN to stop continual diffs

FEATURES:

 * **New DataSource:** `aws_route_table` ([#10301](https://github.com/hashicorp/terraform/issues/10301))
 * **New Interpolation Function:** `timestamp` ([#10475](https://github.com/hashicorp/terraform/issues/10475))

IMPROVEMENTS:

 * core: Plan will show deposed-only destroys for create-before-destroy resources. ([#10404](https://github.com/hashicorp/terraform/issues/10404))
 * provider/aws: Enforced kms_key_* attributes to be ARNs ([#10356](https://github.com/hashicorp/terraform/issues/10356))
 * provider/aws: IPv6 Support To Cloudfront ([#10332](https://github.com/hashicorp/terraform/issues/10332))
 * provider/aws: Support import of aws_iam_instance_profile ([#10436](https://github.com/hashicorp/terraform/issues/10436))
 * provider/aws: Increase `aws_emr_cluster` timeout ([#10444](https://github.com/hashicorp/terraform/issues/10444))
 * provider/aws: Support Automatic Rollback of CodeDeploy deployments and CloudWatch Alarms for a Deployment Group ([#9039](https://github.com/hashicorp/terraform/issues/9039))
 * provider/aws: Allow importing of aws_iam_role, aws_iam_role_policy and aws_iam_policy ([#9398](https://github.com/hashicorp/terraform/issues/9398))
 * provider/aws: Added s3 bucket region attribute management ([#10482](https://github.com/hashicorp/terraform/issues/10482))
 * provider/azurerm: support import of routes, fix route_table ([#10389](https://github.com/hashicorp/terraform/issues/10389))
 * provider/azurerm: create common schema for location field, add diff suppress ([#10409](https://github.com/hashicorp/terraform/issues/10409))
 * provider/github: supports importing resources ([#10382](https://github.com/hashicorp/terraform/issues/10382))
 * provider/postgresql: Added 'connect_timeout' argument to provider 'postgresql' ([#10380](https://github.com/hashicorp/terraform/issues/10380))
 * provider/cloudstack: Support using secondary IP addresses with the `cloudstack_static_nat` resource ([#10420](https://github.com/hashicorp/terraform/issues/10420))
 * provider/google: Added support for session affinity to compute_backend_service ([#10387](https://github.com/hashicorp/terraform/issues/10387))
 * provider/google: Projects are now importable ([#10469](https://github.com/hashicorp/terraform/issues/10469))

BUG FIXES:

  * core: Changed attribute console output shows up on Windows. ([#10417](https://github.com/hashicorp/terraform/issues/10417))
  * core: Destroying deposed resources in create before destroy waits until the creation step of its specific index. (0.8 regression) ([#10416](https://github.com/hashicorp/terraform/issues/10416))
  * core: Certain invalid configurations will no longer print "illegal". ([#10448](https://github.com/hashicorp/terraform/issues/10448))
  * core: Fix a crash that could occur when multiple deposed instances exist. ([#10504](https://github.com/hashicorp/terraform/issues/10504))
  * command/console: variable access works ([#10446](https://github.com/hashicorp/terraform/issues/10446))
  * provider/aws: Do not return a root device for instance store backed AMIs. ([#9483](https://github.com/hashicorp/terraform/issues/9483))
  * provider/aws: resource_aws_opsworks_application does not accept document_root parameter ([#10477](https://github.com/hashicorp/terraform/issues/10477))
  * provider/aws: bug fix when specifying level on aws_opsworks_permission ([#10394](https://github.com/hashicorp/terraform/issues/10394))

## 0.8.0-rc1 (November 23, 2016)

BASED ON: 0.7.13 (includes any changes up to that point as well)

**Please read prior beta notes, as those are also included. The 0.8 changes
will be coalesced for a 0.8 final, but will remain separate for the pre-release
period.**

BACKWARDS INCOMPATIBILITIES / NOTES:

 * The chef provider now accepts `key_material` as an alternative to `private_key_pem`. The `private_key_pem` attribute will be deprecated in a future release
 * The `template_file` resource no longer accepts a direct file path for the `template` attribute. You may either specify a path wrapped in a `file` function or specify a file path with the `filepath` attribute. This was deprecated during 0.7.x.

FEATURES:
 * core: allow outputs to have descriptions ([#9722](https://github.com/hashicorp/terraform/issues/9722))
 * state/azure: support passing of lease ID when writing storage blob ([#10115](https://github.com/hashicorp/terraform/issues/10115))
 * **New Resource:** `aws_ebs_snapshot` ([#10017](https://github.com/hashicorp/terraform/issues/10017))
 * **New Resource:** `openstack_blockstorage_volume_attach_v2` ([#10259](https://github.com/hashicorp/terraform/issues/10259))
 * **New Resource:** `openstack_compute_volume_attach_v2` ([#10260](https://github.com/hashicorp/terraform/issues/10260))
 * **New Data Source:** `aws_ebs_snapshot` ([#10017](https://github.com/hashicorp/terraform/issues/10017))
 * The `import` command can now specify a provider alias to use. ([#10310](https://github.com/hashicorp/terraform/issues/10310))

IMPROVEMENTS:

 * provider/aws: Addition of suspended_processes to aws_autoscaling_group ([#10096](https://github.com/hashicorp/terraform/issues/10096))
 * provider/aws: added auto_minor_version_upgrade on aws_rds_cluster_insstance ([#10284](https://github.com/hashicorp/terraform/issues/10284))
 * provider/aws: Add JSON validation to the aws_iam_policy resource ([#10239](https://github.com/hashicorp/terraform/issues/10239))
 * provider/azurerm: enable import of more resources ([#10195](https://github.com/hashicorp/terraform/issues/10195))
 * provider/chef: Migrate Chef to use KEY_MATERIAL rather than using a Pem file ([#10105](https://github.com/hashicorp/terraform/issues/10105))
 * provider/docker: authentication via values instead of files ([#10151](https://github.com/hashicorp/terraform/issues/10151))
 * provider/google: Add Service Accounts resource ([#9946](https://github.com/hashicorp/terraform/issues/9946))
 * provider/nomad: Update to support Nomad 0.5.0
 * provider/openstack: Add Swauth/Swift Authentication ([#9943](https://github.com/hashicorp/terraform/issues/9943))
 * state/remote/swift: Add support for versioning state file in swift and expiring versioned state ([#10055](https://github.com/hashicorp/terraform/issues/10055))

BUG FIXES:

 * core: Catch parse errors for null characters mid-file ([#9134](https://github.com/hashicorp/terraform/issues/9134))
 * core: escape sequence for " works (0.8 beta regression) ([#10236](https://github.com/hashicorp/terraform/issues/10236))
 * core: Terraform starts on Windows (0.8 beta2 regression) ([#10266](https://github.com/hashicorp/terraform/issues/10266))
 * core: Remove extra dot from state command backup files ([#10300](https://github.com/hashicorp/terraform/issues/10300))
 * core: Validate data sources do not have provisioners ([#10318](https://github.com/hashicorp/terraform/issues/10318))
 * core: Disable checkpoint settings take effect ([#10206](https://github.com/hashicorp/terraform/issues/10206))
 * provider/aws: Skip VPC endpoint routes when removing default route table's routes ([#10303](https://github.com/hashicorp/terraform/issues/10303))
 * provider/azurerm: Prevent null reference when reading boot_diagnostics settings in azurerm_virtual_machine ([#10283](https://github.com/hashicorp/terraform/issues/10283))
 * provider/template: No file path error when setting template to `/` ([#10297](https://github.com/hashicorp/terraform/issues/10297))

PLUGIN CHANGES:

 * The protocol version has been incremented, requiring all plugins for
   0.8 to be built with 0.8 sources (or newer). This should only require
   a simple recompile for compatibility.

## 0.8.0-beta2 (November 16, 2016)

BASED ON: 0.7.11 (includes any changes up to that point as well)

**Please read prior beta notes, as those are also included. The 0.8 changes
will be coalesced for a 0.8 final, but will remain separate for the pre-release
period.**

BACKWARDS INCOMPATIBILITIES / NOTES:

 * Math operators now follow the standard order of operations: *, /, % followed
   by +, -. See the updated interpolation docs for more information. You can
   continue to force ordering with parentheses.

FEATURES:

 * **New command:** `terraform console`, an interactive console for experimenting
   with and using interpolations. ([#10093](https://github.com/hashicorp/terraform/issues/10093))
 * **Terraform version requirement in configuration.** You can now specify
   a Terraform version requirement in configuration and modules. ([#10080](https://github.com/hashicorp/terraform/issues/10080))
 * **`depends_on` can reference modules.** This allows a resource or output
   to depend on everything within a module. ([#10076](https://github.com/hashicorp/terraform/issues/10076))
 * **`output` supports `depends_on`.** This is useful when the output depends
   on a certain ordering to happen that can't be represented with interpolations.
   ([#10072](https://github.com/hashicorp/terraform/issues/10072))

## 0.8.0-beta1 (November 11, 2016)

BASED ON: 0.7.10 (includes any changes up to that point as well)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * `template_file` _inline_ templates must escape their variable usage. What
      was previously `${foo}` must now be `$${foo}`. Note that this is only
      for _inline_ templates. Templates read from files are unchanged. ([#9698](https://github.com/hashicorp/terraform/issues/9698))
 * Escape sequences used to require double-escaping when used within interpolations.
      You now must only escape once (which is the expected/typical behavior).
      For example: `${replace(var.foo, "\\", "\\\\")}` is correct. Before,
      that would cause very strange behavior. However, this may break existing
      configurations which found a level of escape sequences to work. Check
      `terraform plan` for incorrect output.

FEATURES:

 * **New provider:** `nomad` ([#9538](https://github.com/hashicorp/terraform/issues/9538))
 * **New provider:** `vault` ([#9158](https://github.com/hashicorp/terraform/issues/9158))
 * The `import` command will now read provider configuration from Terraform
   configuration files (including loading tfvars files and so on). ([#9809](https://github.com/hashicorp/terraform/issues/9809))
 * Providers and resources are now notified by Terraform core to "stop" when
   an interrupt is received, allowing resources to gracefully exit much, much
   faster. ([#9607](https://github.com/hashicorp/terraform/issues/9607))

IMPROVEMENTS:

  * core: Human-friendly error when a computed count is used. ([#10060](https://github.com/hashicorp/terraform/issues/10060))
  * helper/schema: only map, list, and set elements that are actually causing
      a resource to destroy/create are marked as "requires new". ([#9613](https://github.com/hashicorp/terraform/issues/9613))

BUG FIXES:

  * core: Escape sequences in interpolations work in every case. ([#8709](https://github.com/hashicorp/terraform/issues/8709))
  * core: Maps in outputs with computed values are no longer removed. ([#9549](https://github.com/hashicorp/terraform/issues/9549))
  * command/fmt: Multiline comments aren't indented every fmt. ([#6524](https://github.com/hashicorp/terraform/issues/6524))

## 0.7.13 (November 23, 2016)

BUG FIXES:

 * core: New graph records dependencies for explicit self references ([#10319](https://github.com/hashicorp/terraform/issues/10319))

## 0.7.12 (November 22, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/cloudstack: `cloudstack_static_nat` has now deprecated `network_id` ([#10204](https://github.com/hashicorp/terraform/issues/10204))

FEATURES:

 * *New Data Source:* `aws_alb_listener` ([#10181](https://github.com/hashicorp/terraform/issues/10181))
 * *New Resource:* `github_label` ([#10213](https://github.com/hashicorp/terraform/issues/10213))

IMPROVEMENTS:

 * core: Experimental feature failures are less verbose. ([#10276](https://github.com/hashicorp/terraform/issues/10276))
 * provider/aws: Add name_prefix to aws_iam_policy ([#10178](https://github.com/hashicorp/terraform/issues/10178))
 * provider/aws: Add ability to select aws_prefix_list data source by name ([#10248](https://github.com/hashicorp/terraform/issues/10248))
 * provider/aws Return service CIDR blocks from aws_vpc_endpoint resource ([#10254](https://github.com/hashicorp/terraform/issues/10254))
 * provider/aws: Added `environment` configuration for AWS Lambda Functions ([#10275](https://github.com/hashicorp/terraform/issues/10275))

BUG FIXES:

 * core: Fix potential crashing race condition on state write ([#10277](https://github.com/hashicorp/terraform/issues/10277))
 * core: Data sources in modules lose their `data.` prefix when moved within the state ([#9996](https://github.com/hashicorp/terraform/issues/9996))
 * provider/aws: Fixed issue with `enable_dns_support` on creation in `aws_vpc` ([#10171](https://github.com/hashicorp/terraform/issues/10171))
 * provider/aws: Add CertificateNotFound retry waiter to aws_alb_listener ([#10180](https://github.com/hashicorp/terraform/issues/10180))
 * provider/aws: Remove IAM user's MFA devices with `force_destroy` ([#10262](https://github.com/hashicorp/terraform/issues/10262))
 * provider/scaleway: improve volume attachment ([#10084](https://github.com/hashicorp/terraform/issues/10084))

## 0.7.11 (November 15, 2016)

FEATURES:

IMPROVEMENTS:

 * provider/aws: Expose RDS DB Instance HostedZoneId attribute ([#10000](https://github.com/hashicorp/terraform/issues/10000))
 * provider/aws: Ignore AWS internal tags ([#7454](https://github.com/hashicorp/terraform/issues/7454))
 * provider/aws: Exposed aws_iam_role create_date attribute ([#10091](https://github.com/hashicorp/terraform/issues/10091))
 * provider/aws: Added aws_api_gateway_api_key created_date & last_updated_date attributes ([#9530](https://github.com/hashicorp/terraform/issues/9530))
 * provider/aws: Added aws_api_gateway_rest_api created_date attribute ([#9532](https://github.com/hashicorp/terraform/issues/9532))
 * provider/aws: Exposed aws_api_gateway_deployment.created_date attribute ([#9534](https://github.com/hashicorp/terraform/issues/9534))
 * provider/aws: Added `retry_duration` to `redshift_configuration` in `kinesis_firehose_delivery_stream` ([#10113](https://github.com/hashicorp/terraform/issues/10113))
 * provider/azurerm: allow updating load balancer sub-resources ([#10016](https://github.com/hashicorp/terraform/issues/10016))
 * provider/openstack: Instance `user_data` will now detect if input is already Base64-encode ([#9966](https://github.com/hashicorp/terraform/issues/9966))

BUG FIXES:

 * core: Fix diff mismatch error on "Destroy: true to false" scenarios. ([#10139](https://github.com/hashicorp/terraform/issues/10139))
 * core: New destroy graph `-target` includes dependencies. ([#10036](https://github.com/hashicorp/terraform/issues/10036))
 * core: New destroy graph creates proper edges through module outputs ([#10068](https://github.com/hashicorp/terraform/issues/10068))
 * core: Fix shadow error when using uuid() ([#10106](https://github.com/hashicorp/terraform/issues/10106))
 * core: Fix an issue where applies with data sources could hang ([#10134](https://github.com/hashicorp/terraform/issues/10134))
 * core: Fix plan operation diff mismatch for computed keys in slices ([#10118](https://github.com/hashicorp/terraform/issues/10118))
 * provider/aws: fix the validation of aws_redshift_cluster database_name ([#10019](https://github.com/hashicorp/terraform/issues/10019))
 * provider/aws: Fix panic in aws_acm_certificate datasource ([#10051](https://github.com/hashicorp/terraform/issues/10051))
 * provider/aws: increase aws_lambda_function timeout ([#10116](https://github.com/hashicorp/terraform/issues/10116))
 * provider/aws: Fixed ES buffering_interval option in `kinesis_firehose_delivery_stream` ([#10112](https://github.com/hashicorp/terraform/issues/10112))

## 0.7.10 (November 9, 2016)

FEATURES:

 * **New Resource:** `azurerm_eventhub` ([#9889](https://github.com/hashicorp/terraform/issues/9889))
 * **New Resource:** `azurerm_virtual_machine_extension` ([#9962](https://github.com/hashicorp/terraform/issues/9962))
 * **Experimental new plan graph:** `terraform plan` is getting a new graph
   creation process for 0.8. This is now available behind a flag `-Xnew-apply`
   (on any command). This will become the default in 0.8. There may still be
   bugs. ([#9973](https://github.com/hashicorp/terraform/issues/9973))

IMPROVEMENTS:

 * provider/aws: Add support for Service Access Security Group in `aws_emr_cluster` ([#9600](https://github.com/hashicorp/terraform/issues/9600))
 * provider/aws: Add Enhanced VPC routing to Redshift ([#9950](https://github.com/hashicorp/terraform/issues/9950))
 * provider/aws: Add key_name_prefix argument to aws_key_pair resource ([#9993](https://github.com/hashicorp/terraform/issues/9993))
 * provider/openstack: Add `value_specs` to `openstack_fw_policy_v1` resource, allowing vendor information ([#9835](https://github.com/hashicorp/terraform/issues/9835))
 * provider/openstack: Add `value_specs` to `openstack_fw_firewall_v1` resource, allowing vendor information ([#9836](https://github.com/hashicorp/terraform/issues/9836))
 * provider/random: The `b64` attribute on `random_id` resources is deprecated, replaced by `b64_url` and `b64_std` ([#9903](https://github.com/hashicorp/terraform/issues/9903))

BUG FIXES:

 * core: Splat variables (`foo.*.bar`) are now ordered by count index for deterministic ordering. ([#9883](https://github.com/hashicorp/terraform/issues/9883))
 * core: Prune orphan outputs (in the config but not in the state). ([#9971](https://github.com/hashicorp/terraform/issues/9971))
 * core: New apply graph doesn't prune module variables as aggressively. ([#9898](https://github.com/hashicorp/terraform/issues/9898))
 * core: New apply graph properly configures providers with aliases. ([#9894](https://github.com/hashicorp/terraform/issues/9894))
 * core: New destroy graph doesn't create edge loops to destroy nodes that reference themselves. ([#9968](https://github.com/hashicorp/terraform/issues/9968))
 * provider/aws: Fix crash when adding EBS volumes to spot fleet request. ([#9857](https://github.com/hashicorp/terraform/issues/9857))
 * provider/aws: Ignore NoSuchEntity error when IAM user does not have login profile ([#9900](https://github.com/hashicorp/terraform/issues/9900))
 * provider/aws: Setting static_routes_only on import of vpn_connection ([#9802](https://github.com/hashicorp/terraform/issues/9802))
 * provider/aws: aws_alb_target_group arn_suffix missing the targetgroup ([#9911](https://github.com/hashicorp/terraform/issues/9911))
 * provider/aws: Fix the validateFunc of aws_elasticache_replication_group ([#9918](https://github.com/hashicorp/terraform/issues/9918))
 * provider/aws: removing toLower when setting aws_db_parameter_group options ([#9820](https://github.com/hashicorp/terraform/issues/9820))
 * provider/aws: Fix panic when passing statuses to aws_acm_certificate ([#9990](https://github.com/hashicorp/terraform/issues/9990))
 * provider/aws: AWS IAM, User and Role allow + in the name ([#9991](https://github.com/hashicorp/terraform/issues/9991))
 * provider/scaleway: retry volume attachment ([#9972](https://github.com/hashicorp/terraform/issues/9972))
 * provider/scaleway: fix `scaleway_image` datasource returning unknown images ([#9899](https://github.com/hashicorp/terraform/issues/9899))
 * provider/google: fix crash when mistakenly configuring disks ([#9942](https://github.com/hashicorp/terraform/issues/9942))

## 0.7.9 (November 4, 2016)

FEATURES:

 * **New Data Source:** `aws_acm_certificate` ([#8359](https://github.com/hashicorp/terraform/issues/8359))
 * **New Resource:** `aws_autoscaling_attachment` ([#9146](https://github.com/hashicorp/terraform/issues/9146))
 * **New Resource:** `postgresql_extension` ([#9210](https://github.com/hashicorp/terraform/issues/9210))

IMPROVEMENTS:

 * core: Improve shadow graph robustness by catching panics during graph evaluation. ([#9852](https://github.com/hashicorp/terraform/issues/9852))
 * provider/aws: Provide the option to skip_destroy on aws_volume_attachment ([#9792](https://github.com/hashicorp/terraform/issues/9792))
 * provider/aws: Allows aws_alb security_groups to be updated ([#9804](https://github.com/hashicorp/terraform/issues/9804))
 * provider/aws: Add the enable_sni attribute for Route53 health checks. ([#9822](https://github.com/hashicorp/terraform/issues/9822))
 * provider/openstack: Add `value_specs` to openstack_fw_rule_v1 resource, allowing vendor information ([#9834](https://github.com/hashicorp/terraform/issues/9834))
 * state/remote/swift: Enable OpenStack Identity/Keystone v3 authentication ([#9769](https://github.com/hashicorp/terraform/issues/9769))
 * state/remote/swift: Now supports all login/config options that the OpenStack Provider supports ([#9777](https://github.com/hashicorp/terraform/issues/9777))

BUG FIXES:

 * core: Provisioners in modules do not crash during `apply` (regression). ([#9846](https://github.com/hashicorp/terraform/issues/9846))
 * core: Computed bool fields with non-bool values will not crash ([#9812](https://github.com/hashicorp/terraform/issues/9812))
 * core: `formatlist` interpolation function accepts an empty list ([#9795](https://github.com/hashicorp/terraform/issues/9795))
 * core: Validate outputs have a name ([#9823](https://github.com/hashicorp/terraform/issues/9823))
 * core: Validate variables have a name ([#9818](https://github.com/hashicorp/terraform/issues/9818))
 * command/apply: If a partial set of required variables are provided with `-var`, ask for the remainder ([#9794](https://github.com/hashicorp/terraform/issues/9794))
 * command/fmt: Multiline strings aren't erroneously indented ([#9859](https://github.com/hashicorp/terraform/issues/9859))
 * provider/aws: Fix issue setting `certificate_upload_date` in `aws_api_gateway_domain_name` ([#9815](https://github.com/hashicorp/terraform/issues/9815))
 * provider/azurerm: allow storage_account resource with name "$root" ([#9813](https://github.com/hashicorp/terraform/issues/9813))
 * provider/google: fix for looking up project image families ([#9243](https://github.com/hashicorp/terraform/issues/9243))
 * provider/openstack: Don't pass `shared` in FWaaS Policy unless it's set ([#9830](https://github.com/hashicorp/terraform/issues/9830))
 * provider/openstack: openstack_fw_firewall_v1 `admin_state_up` should default to true ([#9832](https://github.com/hashicorp/terraform/issues/9832))

PLUGIN CHANGES:

  * Fields in resources can now have both `Optional` and `ConflictsWith` ([#9825](https://github.com/hashicorp/terraform/issues/9825))

## 0.7.8 (November 1, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/openstack: The OpenStack provider has switched to the new Gophercloud SDK.
   No front-facing changes were made, but please be aware that there might be bugs.
   Please report any if found.
 * `archive_file` is now a data source, instead of a resource ([#8492](https://github.com/hashicorp/terraform/issues/8492))

FEATURES:

 * **Experimental new apply graph:** `terraform apply` is getting a new graph
   creation process for 0.8. This is now available behind a flag `-Xnew-apply`
   (on any command). This will become the default in 0.8. There may still be
   bugs. ([#9388](https://github.com/hashicorp/terraform/issues/9388))
 * **Experimental new destroy graph:** `terraform destroy` is also getting
   a new graph creation process for 0.8. This is now available behind a flag
   `-Xnew-destroy`. This will become the default in 0.8. ([#9527](https://github.com/hashicorp/terraform/issues/9527))
 * **New Provider:** `pagerduty` ([#9022](https://github.com/hashicorp/terraform/issues/9022))
 * **New Resource:** `aws_iam_user_login_profile` ([#9605](https://github.com/hashicorp/terraform/issues/9605))
 * **New Resource:** `aws_waf_ipset` ([#8852](https://github.com/hashicorp/terraform/issues/8852))
 * **New Resource:** `aws_waf_rule` ([#8852](https://github.com/hashicorp/terraform/issues/8852))
 * **New Resource:** `aws_waf_web_acl` ([#8852](https://github.com/hashicorp/terraform/issues/8852))
 * **New Resource:** `aws_waf_byte_match_set` ([#9681](https://github.com/hashicorp/terraform/issues/9681))
 * **New Resource:** `aws_waf_size_constraint_set` ([#9689](https://github.com/hashicorp/terraform/issues/9689))
 * **New Resource:** `aws_waf_sql_injection_match_set` ([#9709](https://github.com/hashicorp/terraform/issues/9709))
 * **New Resource:** `aws_waf_xss_match_set` ([#9710](https://github.com/hashicorp/terraform/issues/9710))
 * **New Resource:** `aws_ssm_activation` ([#9111](https://github.com/hashicorp/terraform/issues/9111))
 * **New Resource:** `azurerm_key_vault` ([#9478](https://github.com/hashicorp/terraform/issues/9478))
 * **New Resource:** `azurerm_storage_share` ([#8674](https://github.com/hashicorp/terraform/issues/8674))
 * **New Resource:** `azurerm_eventhub_namespace` ([#9297](https://github.com/hashicorp/terraform/issues/9297))
 * **New Resource:** `cloudstack_security_group` ([#9103](https://github.com/hashicorp/terraform/issues/9103))
 * **New Resource:** `cloudstack_security_group_rule` ([#9645](https://github.com/hashicorp/terraform/issues/9645))
 * **New Resource:** `cloudstack_private_gateway` ([#9637](https://github.com/hashicorp/terraform/issues/9637))
 * **New Resource:** `cloudstack_static_route` ([#9637](https://github.com/hashicorp/terraform/issues/9637))
 * **New DataSource:** `aws_ebs_volume` ([#9753](https://github.com/hashicorp/terraform/issues/9753))
 * **New DataSource:** `aws_prefix_list` ([#9566](https://github.com/hashicorp/terraform/issues/9566))
 * **New DataSource:** `aws_security_group` ([#9604](https://github.com/hashicorp/terraform/issues/9604))
 * **New DataSource:** `azurerm_client_config` ([#9478](https://github.com/hashicorp/terraform/issues/9478))
 * **New Interpolation Function:** `ceil` ([#9692](https://github.com/hashicorp/terraform/issues/9692))
 * **New Interpolation Function:** `floor` ([#9692](https://github.com/hashicorp/terraform/issues/9692))
 * **New Interpolation Function:** `min` ([#9692](https://github.com/hashicorp/terraform/issues/9692))
 * **New Interpolation Function:** `max` ([#9692](https://github.com/hashicorp/terraform/issues/9692))
 * **New Interpolation Function:** `title` ([#9087](https://github.com/hashicorp/terraform/issues/9087))
 * **New Interpolation Function:** `zipmap` ([#9627](https://github.com/hashicorp/terraform/issues/9627))

IMPROVEMENTS:

 * provider/aws: No longer require `route_table_ids` list in `aws_vpc_endpoint` resources ([#9357](https://github.com/hashicorp/terraform/issues/9357))
 * provider/aws: Allow `description` in `aws_redshift_subnet_group` to be modified ([#9515](https://github.com/hashicorp/terraform/issues/9515))
 * provider/aws: Add tagging support to aws_redshift_subnet_group ([#9504](https://github.com/hashicorp/terraform/issues/9504))
 * provider/aws: Add validation to IAM User and Group Name ([#9584](https://github.com/hashicorp/terraform/issues/9584))
 * provider/aws: Add Ability To Enable / Disable ALB AccessLogs ([#9290](https://github.com/hashicorp/terraform/issues/9290))
 * provider/aws: Add support for `AutoMinorVersionUpgrade` to aws_elasticache_replication_group resource. ([#9657](https://github.com/hashicorp/terraform/issues/9657))
 * provider/aws: Fix import of RouteTable with destination prefixes ([#9686](https://github.com/hashicorp/terraform/issues/9686))
 * provider/aws: Add support for reference_name to aws_route53_health_check ([#9737](https://github.com/hashicorp/terraform/issues/9737))
 * provider/aws: Expose ARN suffix on ALB Target Group ([#9734](https://github.com/hashicorp/terraform/issues/9734))
 * provider/azurerm: add account_kind and access_tier to storage_account ([#9408](https://github.com/hashicorp/terraform/issues/9408))
 * provider/azurerm: write load_balanacer attributes to network_interface_card hash ([#9207](https://github.com/hashicorp/terraform/issues/9207))
 * provider/azurerm: Add disk_size_gb param to VM storage_os_disk ([#9200](https://github.com/hashicorp/terraform/issues/9200))
 * provider/azurerm: support importing of subnet resource ([#9646](https://github.com/hashicorp/terraform/issues/9646))
 * provider/azurerm: Add support for *all* of the Azure regions e.g. Germany, China and Government ([#9765](https://github.com/hashicorp/terraform/issues/9765))
 * provider/digitalocean: Allow resizing DigitalOcean Droplets without increasing disk size. ([#9573](https://github.com/hashicorp/terraform/issues/9573))
 * provider/google: enhance service scope list ([#9442](https://github.com/hashicorp/terraform/issues/9442))
 * provider/google Change default MySQL instance version to 5.6 ([#9674](https://github.com/hashicorp/terraform/issues/9674))
 * provider/google Support MySQL 5.7 instances ([#9673](https://github.com/hashicorp/terraform/issues/9673))
 * provider/google: Add support for using source_disk to google_compute_image ([#9614](https://github.com/hashicorp/terraform/issues/9614))
 * provider/google: Add support for default-internet-gateway alias for google_compute_route ([#9676](https://github.com/hashicorp/terraform/issues/9676))
 * provider/openstack: Added value_specs to openstack_networking_port_v2, allowing vendor information ([#9551](https://github.com/hashicorp/terraform/issues/9551))
 * provider/openstack: Added value_specs to openstack_networking_floatingip_v2, allowing vendor information ([#9552](https://github.com/hashicorp/terraform/issues/9552))
 * provider/openstack: Added value_specs to openstack_compute_keypair_v2, allowing vendor information ([#9554](https://github.com/hashicorp/terraform/issues/9554))
 * provider/openstack: Allow any protocol in openstack_fw_rule_v1 ([#9617](https://github.com/hashicorp/terraform/issues/9617))
 * provider/openstack: expose LoadBalancer v2 VIP Port ID ([#9727](https://github.com/hashicorp/terraform/issues/9727))
 * provider/openstack: Openstack Provider enhancements including environment variables ([#9725](https://github.com/hashicorp/terraform/issues/9725))
 * provider/scaleway: update sdk for ams1 region ([#9687](https://github.com/hashicorp/terraform/issues/9687))
 * provider/scaleway: server volume property ([#9695](https://github.com/hashicorp/terraform/issues/9695))

BUG FIXES:

 * core: Resources suffixed with 'panic' won't falsely trigger crash detection. ([#9395](https://github.com/hashicorp/terraform/issues/9395))
 * core: Validate lifecycle options don't contain interpolations. ([#9576](https://github.com/hashicorp/terraform/issues/9576))
 * core: Tainted resources will not process `ignore_changes`. ([#7855](https://github.com/hashicorp/terraform/issues/7855))
 * core: Boolean looking values passed in via `-var` no longer cause type errors. ([#9642](https://github.com/hashicorp/terraform/issues/9642))
 * core: Computed primitives in certain cases no longer cause diff mismatch errors. ([#9618](https://github.com/hashicorp/terraform/issues/9618))
 * core: Empty arrays for list vars in JSON work ([#8886](https://github.com/hashicorp/terraform/issues/8886))
 * core: Boolean types in tfvars work propertly ([#9751](https://github.com/hashicorp/terraform/issues/9751))
 * core: Deposed resource destruction is accounted for properly in `apply` counts. ([#9731](https://github.com/hashicorp/terraform/issues/9731))
 * core: Check for graph cycles on resource expansion to catch cycles between self-referenced resources. ([#9728](https://github.com/hashicorp/terraform/issues/9728))
 * core: `prevent_destroy` prevents decreasing count ([#9707](https://github.com/hashicorp/terraform/issues/9707))
 * core: removed optional items will trigger "requires new" if necessary ([#9699](https://github.com/hashicorp/terraform/issues/9699))
 * command/apply: `-backup` and `-state-out` work with plan files ([#9706](https://github.com/hashicorp/terraform/issues/9706))
 * command/fmt: Cleaner formatting for multiline standalone comments above resources
 * command/validate: respond to `--help` ([#9660](https://github.com/hashicorp/terraform/issues/9660))
 * provider/archive: Converting to datasource. ([#8492](https://github.com/hashicorp/terraform/issues/8492))
 * provider/aws: Fix issue importing AWS Instances and setting the correct `associate_public_ip_address` value ([#9453](https://github.com/hashicorp/terraform/issues/9453))
 * provider/aws: Fix issue with updating ElasticBeanstalk environment variables ([#9259](https://github.com/hashicorp/terraform/issues/9259))
 * provider/aws: Allow zero value for `scaling_adjustment` in `aws_autoscaling_policy` when using `SimpleScaling` ([#8893](https://github.com/hashicorp/terraform/issues/8893))
 * provider/aws: Increase ECS service drain timeout ([#9521](https://github.com/hashicorp/terraform/issues/9521))
 * provider/aws: Remove VPC Endpoint from state if it's not found ([#9561](https://github.com/hashicorp/terraform/issues/9561))
 * provider/aws: Delete Loging Profile from IAM User on force_destroy ([#9583](https://github.com/hashicorp/terraform/issues/9583))
 * provider/aws: Exposed aws_api_gw_domain_name.certificate_upload_date attribute ([#9533](https://github.com/hashicorp/terraform/issues/9533))
 * provider/aws: fix aws_elasticache_replication_group for Redis in cluster mode ([#9601](https://github.com/hashicorp/terraform/issues/9601))
 * provider/aws: Validate regular expression passed via the ami data_source `name_regex` attribute. ([#9622](https://github.com/hashicorp/terraform/issues/9622))
 * provider/aws: Bug fix for NoSuckBucket on Destroy of aws_s3_bucket_policy ([#9641](https://github.com/hashicorp/terraform/issues/9641))
 * provider/aws: Refresh aws_autoscaling_schedule from state on 404 ([#9659](https://github.com/hashicorp/terraform/issues/9659))
 * provider/aws: Allow underscores in IAM user and group names ([#9684](https://github.com/hashicorp/terraform/issues/9684))
 * provider/aws: aws_ami: handle deletion of AMIs ([#9721](https://github.com/hashicorp/terraform/issues/9721))
 * provider/aws: Fix aws_route53_record alias perpetual diff ([#9704](https://github.com/hashicorp/terraform/issues/9704))
 * provider/aws: Allow `active` state while waiting for the VPC Peering Connection. ([#9754](https://github.com/hashicorp/terraform/issues/9754))
 * provider/aws: Normalize all-principals wildcard in `aws_iam_policy_document` ([#9720](https://github.com/hashicorp/terraform/issues/9720))
 * provider/azurerm: Fix Azure RM loadbalancer rules validation ([#9468](https://github.com/hashicorp/terraform/issues/9468))
 * provider/azurerm: Fix servicebus_topic values when using the Update func to stop perpetual diff ([#9323](https://github.com/hashicorp/terraform/issues/9323))
 * provider/azurerm: lower servicebus_topic max size to Azure limit ([#9649](https://github.com/hashicorp/terraform/issues/9649))
 * provider/azurerm: Fix VHD deletion when VM and Storage account are in separate resource groups ([#9631](https://github.com/hashicorp/terraform/issues/9631))
 * provider/azurerm: Guard against panic when importing arm_virtual_network ([#9739](https://github.com/hashicorp/terraform/issues/9739))
 * provider/azurerm: fix sql_database resource reading tags ([#9767](https://github.com/hashicorp/terraform/issues/9767))
 * provider/cloudflare: update client library to stop connection closed issues ([#9715](https://github.com/hashicorp/terraform/issues/9715))
 * provider/consul: Change to consul_service resource to introduce a `service_id` parameter ([#9366](https://github.com/hashicorp/terraform/issues/9366))
 * provider/datadog: Ignore float/int diffs on thresholds ([#9466](https://github.com/hashicorp/terraform/issues/9466))
 * provider/docker: Fixes for docker_container host object and documentation ([#9367](https://github.com/hashicorp/terraform/issues/9367))
 * provider/scaleway improve the performance of server deletion ([#9491](https://github.com/hashicorp/terraform/issues/9491))
 * provider/scaleway: fix scaleway_volume_attachment with count > 1 ([#9493](https://github.com/hashicorp/terraform/issues/9493))


## 0.7.7 (October 18, 2016)

FEATURES:

 * **New Data Source:** `scaleway_bootsscript`. ([#9386](https://github.com/hashicorp/terraform/issues/9386))
 * **New Data Source:** `scaleway_image`. ([#9386](https://github.com/hashicorp/terraform/issues/9386))

IMPROVEMENTS:

 * core: When the environment variable TF_LOG_PATH is specified, debug logs are now appended to the specified file instead of being truncated. ([#9440](https://github.com/hashicorp/terraform/pull/9440))
 * provider/aws: Expose ARN for `aws_lambda_alias`. ([#9390](https://github.com/hashicorp/terraform/issues/9390))
 * provider/aws: Add support for AWS US East (Ohio) region. ([#9414](https://github.com/hashicorp/terraform/issues/9414))
 * provider/scaleway: `scaleway_ip`, `scaleway_security_group`, `scalway_server` and `scaleway_volume` resources can now be imported. ([#9387](https://github.com/hashicorp/terraform/issues/9387))

BUG FIXES:

 * core: List and map indexes support arithmetic. ([#9372](https://github.com/hashicorp/terraform/issues/9372))
 * core: List and map indexes are implicitly converted to the correct type if possible. ([#9372](https://github.com/hashicorp/terraform/issues/9372))
 * provider/aws: Read back `associate_public_ip_address` in `aws_launch_configuration` resources to enable importing. ([#9399](https://github.com/hashicorp/terraform/issues/9399))
 * provider/aws: Remove `aws_route` resources from state if their associated `aws_route_table` has been removed. ([#9431](https://github.com/hashicorp/terraform/issues/9431))
 * provider/azurerm: Load balancer resources now have their `id` attribute set to the resource URI instead of the load balancer URI. ([#9401](https://github.com/hashicorp/terraform/issues/9401))
 * provider/google: Fix a bug causing a crash when migrating `google_compute_target_pool` resources from 0.6.x releases. ([#9370](https://github.com/hashicorp/terraform/issues/9370))

## 0.7.6 (October 14, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:
 * `azurerm_virtual_machine` has deprecated the use of `diagnostics_profile` in favour of `boot_diagnostics`. ([#9122](https://github.com/hashicorp/terraform/issues/9122))
 * The deprecated `key_file` and `bastion_key_file` arguments to Provisioner Connections have been removed ([#9340](https://github.com/hashicorp/terraform/issues/9340))

FEATURES:
 * **New Data Source:** `aws_billing_service_account` ([#8701](https://github.com/hashicorp/terraform/issues/8701))
 * **New Data Source:** `aws_availability_zone` ([#6819](https://github.com/hashicorp/terraform/issues/6819))
 * **New Data Source:** `aws_region` ([#6819](https://github.com/hashicorp/terraform/issues/6819))
 * **New Data Source:** `aws_subnet` ([#6819](https://github.com/hashicorp/terraform/issues/6819))
 * **New Data Source:** `aws_vpc` ([#6819](https://github.com/hashicorp/terraform/issues/6819))
 * **New Resource:** `azurerm_lb` ([#9199](https://github.com/hashicorp/terraform/issues/9199))
 * **New Resource:** `azurerm_lb_backend_address_pool` ([#9199](https://github.com/hashicorp/terraform/issues/9199))
 * **New Resource:** `azurerm_lb_nat_rule` ([#9199](https://github.com/hashicorp/terraform/issues/9199))
 * **New Resource:** `azurerm_lb_nat_pool` ([#9199](https://github.com/hashicorp/terraform/issues/9199))
 * **New Resource:** `azurerm_lb_probe` ([#9199](https://github.com/hashicorp/terraform/issues/9199))
 * **New Resource:** `azurerm_lb_rule` ([#9199](https://github.com/hashicorp/terraform/issues/9199))
 * **New Resource:** `github_repository` ([#9327](https://github.com/hashicorp/terraform/issues/9327))

IMPROVEMENTS:
 * core-validation: create validation package to provide common validation functions ([#8103](https://github.com/hashicorp/terraform/issues/8103))
 * provider/aws: Support Import of OpsWorks Custom Layers ([#9252](https://github.com/hashicorp/terraform/issues/9252))
 * provider/aws: Automatically constructed ARNs now support partitions other than `aws`, allowing operation with `aws-cn` and `aws-us-gov` ([#9273](https://github.com/hashicorp/terraform/issues/9273))
 * provider/aws: Retry setTags operation for EC2 resources ([#7890](https://github.com/hashicorp/terraform/issues/7890))
 * provider/aws: Support refresh of EC2 instance `user_data` ([#6736](https://github.com/hashicorp/terraform/issues/6736))
 * provider/aws: Poll to confirm delete of `resource_aws_customer_gateway` ([#9346](https://github.com/hashicorp/terraform/issues/9346))
 * provider/azurerm: expose default keys for `servicebus_namespace` ([#9242](https://github.com/hashicorp/terraform/issues/9242))
 * provider/azurerm: add `enable_blob_encryption` to `azurerm_storage_account` resource ([#9233](https://github.com/hashicorp/terraform/issues/9233))
 * provider/azurerm: set `resource_group_name` on resource import across the provider ([#9073](https://github.com/hashicorp/terraform/issues/9073))
 * provider/azurerm: `azurerm_cdn_profile` resources can now be imported ([#9306](https://github.com/hashicorp/terraform/issues/9306))
 * provider/datadog: add support for Datadog dashboard "type" and "style" options ([#9228](https://github.com/hashicorp/terraform/issues/9228))
 * provider/scaleway: `region` is now supported for provider configuration

BUG FIXES:
 * core: Local state can now be refreshed when no resources exist ([#7320](https://github.com/hashicorp/terraform/issues/7320))
 * core: Orphaned nested (depth 2+) modules will inherit provider configs ([#9318](https://github.com/hashicorp/terraform/issues/9318))
 * core: Fix crash when a map key contains an interpolation function ([#9282](https://github.com/hashicorp/terraform/issues/9282))
 * core: Numeric variables values were incorrectly converted to numbers ([#9263](https://github.com/hashicorp/terraform/issues/9263))
 * core: Fix input and output of map variables from HCL ([#9268](https://github.com/hashicorp/terraform/issues/9268))
 * core: Crash when interpolating a map value with a function in the key ([#9282](https://github.com/hashicorp/terraform/issues/9282))
 * core: Crash when copying a nil value in an InstanceState ([#9356](https://github.com/hashicorp/terraform/issues/9356))
 * command/fmt: Bare comment groups no longer have superfluous newlines
 * command/fmt: Leading comments on list items are formatted properly
 * provider/aws: Return correct AMI image when `most_recent` is set to `true`. ([#9277](https://github.com/hashicorp/terraform/issues/9277))
 * provider/aws: Fix issue with diff on import of `aws_eip` in EC2 Classic ([#9009](https://github.com/hashicorp/terraform/issues/9009))
 * provider/aws: Handle EC2 tags related errors in CloudFront Distribution resource. ([#9298](https://github.com/hashicorp/terraform/issues/9298))
 * provider/aws: Fix cause error when using `etag` and `kms_key_id` with `aws_s3_bucket_object` ([#9168](https://github.com/hashicorp/terraform/issues/9168))
 * provider/aws: Fix issue reassigning EIP instances appropriately ([#7686](https://github.com/hashicorp/terraform/issues/7686))
 * provider/azurerm: removing resources from state when the API returns a 404 for them ([#8859](https://github.com/hashicorp/terraform/issues/8859))
 * provider/azurerm: Fixed a panic in `azurerm_virtual_machine` when using `diagnostic_profile` ([#9122](https://github.com/hashicorp/terraform/issues/9122))

## 0.7.5 (October 6, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:
 * `tls_cert_request` is now a managed resource instead of a data source, restoring the pre-Terraform 0.7 behaviour ([#9035](https://github.com/hashicorp/terraform/issues/9035))

FEATURES:
 * **New Provider:** `bitbucket` ([#7405](https://github.com/hashicorp/terraform/issues/7405))
 * **New Resource:** `aws_api_gateway_client_certificate` ([#8775](https://github.com/hashicorp/terraform/issues/8775))
 * **New Resource:** `azurerm_servicebus_topic` ([#9151](https://github.com/hashicorp/terraform/issues/9151))
 * **New Resource:** `azurerm_servicebus_subscription` ([#9185](https://github.com/hashicorp/terraform/issues/9185))
 * **New Resource:** `aws_emr_cluster` ([#9106](https://github.com/hashicorp/terraform/issues/9106))
 * **New Resource:** `aws_emr_instance_group` ([#9106](https://github.com/hashicorp/terraform/issues/9106))

IMPROVEMENTS:
 * helper/schema: Adding of MinItems as a validation to Lists and Maps ([#9216](https://github.com/hashicorp/terraform/issues/9216))
 * provider/aws: Add JSON validation to the `aws_cloudwatch_event_rule` resource ([#8897](https://github.com/hashicorp/terraform/issues/8897))
 * provider/aws: S3 bucket policies are imported as separate resources ([#8915](https://github.com/hashicorp/terraform/issues/8915))
 * provider/aws: S3 bucket policies can now be removed via the `aws_s3_bucket` resource ([#8915](https://github.com/hashicorp/terraform/issues/8915))
 * provider/aws: Added a `cluster_address` attribute to aws elasticache ([#8935](https://github.com/hashicorp/terraform/issues/8935))
 * provider/aws: Add JSON validation to the `aws_elasticsearch_domain resource`. ([#8898](https://github.com/hashicorp/terraform/issues/8898))
 * provider/aws: Add JSON validation to the `aws_kms_key resource`. ([#8900](https://github.com/hashicorp/terraform/issues/8900))
 * provider/aws: Add JSON validation to the `aws_s3_bucket_policy resource`. ([#8901](https://github.com/hashicorp/terraform/issues/8901))
 * provider/aws: Add JSON validation to the `aws_sns_topic resource`. ([#8902](https://github.com/hashicorp/terraform/issues/8902))
 * provider/aws: Add JSON validation to the `aws_sns_topic_policy resource`. ([#8903](https://github.com/hashicorp/terraform/issues/8903))
 * provider/aws: Add JSON validation to the `aws_sqs_queue resource`. ([#8904](https://github.com/hashicorp/terraform/issues/8904))
 * provider/aws: Add JSON validation to the `aws_sqs_queue_policy resource`. ([#8905](https://github.com/hashicorp/terraform/issues/8905))
 * provider/aws: Add JSON validation to the `aws_vpc_endpoint resource`. ([#8906](https://github.com/hashicorp/terraform/issues/8906))
 * provider/aws: Update `aws_cloudformation_stack` data source with new helper function. ([#8907](https://github.com/hashicorp/terraform/issues/8907))
 * provider/aws: Add JSON validation to the `aws_s3_bucket` resource. ([#8908](https://github.com/hashicorp/terraform/issues/8908))
 * provider/aws: Add support for `cloudwatch_logging_options` to Firehose Delivery Streams ([#8671](https://github.com/hashicorp/terraform/issues/8671))
 * provider/aws: Add HTTP/2 support via the http_version parameter to CloudFront distribution ([#8777](https://github.com/hashicorp/terraform/issues/8777))
 * provider/aws: Add `query_string_cache_keys` to allow for selective caching of CloudFront keys ([#8777](https://github.com/hashicorp/terraform/issues/8777))
 * provider/aws: Support Import `aws_elasticache_cluster` ([#9010](https://github.com/hashicorp/terraform/issues/9010))
 * provider/aws: Add support for tags to `aws_cloudfront_distribution` ([#9011](https://github.com/hashicorp/terraform/issues/9011))
 * provider/aws: Support Import `aws_opsworks_stack` ([#9124](https://github.com/hashicorp/terraform/issues/9124))
 * provider/aws: Support Import `aws_elasticache_replication_groups` ([#9140](https://github.com/hashicorp/terraform/issues/9140))
 * provider/aws: Add new aws api-gateway integration types ([#9213](https://github.com/hashicorp/terraform/issues/9213))
 * provider/aws: Import `aws_db_event_subscription` ([#9220](https://github.com/hashicorp/terraform/issues/9220))
 * provider/azurerm: Add normalizeJsonString and validateJsonString functions ([#8909](https://github.com/hashicorp/terraform/issues/8909))
 * provider/azurerm: Support AzureRM Sql Database DataWarehouse ([#9196](https://github.com/hashicorp/terraform/issues/9196))
 * provider/openstack: Use proxy environment variables for communication with services ([#8948](https://github.com/hashicorp/terraform/issues/8948))
 * provider/vsphere: Adding `detach_unknown_disks_on_delete` flag for VM resource ([#8947](https://github.com/hashicorp/terraform/issues/8947))
 * provisioner/chef: Add `skip_register` attribute to allow skipping the registering steps ([#9127](https://github.com/hashicorp/terraform/issues/9127))

BUG FIXES:
 * core: Fixed variables not being in scope for destroy -target on modules ([#9021](https://github.com/hashicorp/terraform/issues/9021))
 * core: Fixed issue that prevented diffs from being properly generated in a specific resource schema scenario ([#8891](https://github.com/hashicorp/terraform/issues/8891))
 * provider/aws: Remove support for `ah` and `esp` literals in Security Group Ingress/Egress rules; you must use the actual protocol number for protocols other than `tcp`, `udp`, `icmp`, or `all` ([#8975](https://github.com/hashicorp/terraform/issues/8975))
 * provider/aws: Do not report drift for effect values differing only by case in AWS policies ([#9139](https://github.com/hashicorp/terraform/issues/9139))
 * provider/aws: VPC ID, Port, Protocol and Name change on aws_alb_target_group will ForceNew resource ([#8989](https://github.com/hashicorp/terraform/issues/8989))
 * provider/aws: Wait for Spot Fleet to drain before removing from state ([#8938](https://github.com/hashicorp/terraform/issues/8938))
 * provider/aws: Fix issue when importing `aws_eip` resources by IP address ([#8970](https://github.com/hashicorp/terraform/issues/8970))
 * provider/aws: Ensure that origin_access_identity is a required value within the CloudFront distribution s3_config block ([#8777](https://github.com/hashicorp/terraform/issues/8777))
 * provider/aws: Corrected Seoul S3 Website Endpoint format ([#9032](https://github.com/hashicorp/terraform/issues/9032))
 * provider/aws: Fix failed remove S3 lifecycle_rule ([#9031](https://github.com/hashicorp/terraform/issues/9031))
 * provider/aws: Fix crashing bug in `aws_ami` data source when using `name_regex` ([#9033](https://github.com/hashicorp/terraform/issues/9033))
 * provider/aws: Fix reading dimensions on cloudwatch alarms ([#9029](https://github.com/hashicorp/terraform/issues/9029))
 * provider/aws: Changing snapshot_identifier on aws_db_instance resource should force… ([#8806](https://github.com/hashicorp/terraform/issues/8806))
 * provider/aws: Refresh AWS EIP association from state when not found ([#9056](https://github.com/hashicorp/terraform/issues/9056))
 * provider/aws: Make encryption in Aurora instances computed-only ([#9060](https://github.com/hashicorp/terraform/issues/9060))
 * provider/aws: Make sure that VPC Peering Connection in a failed state returns an error. ([#9038](https://github.com/hashicorp/terraform/issues/9038))
 * provider/aws: guard against aws_route53_record delete panic ([#9049](https://github.com/hashicorp/terraform/issues/9049))
 * provider/aws: aws_db_option_group flattenOptions failing due to missing values ([#9052](https://github.com/hashicorp/terraform/issues/9052))
 * provider/aws: Add retry logic to the aws_ecr_repository delete func ([#9050](https://github.com/hashicorp/terraform/issues/9050))
 * provider/aws: Modifying the parameter_group_name of aws_elasticache_replication_group caused a panic ([#9101](https://github.com/hashicorp/terraform/issues/9101))
 * provider/aws: Fix issue with updating ELB subnets for subnets in the same AZ ([#9131](https://github.com/hashicorp/terraform/issues/9131))
 * provider/aws: aws_route53_record alias refresh manually updated record ([#9125](https://github.com/hashicorp/terraform/issues/9125))
 * provider/aws: Fix issue detaching volumes that were already detached ([#9023](https://github.com/hashicorp/terraform/issues/9023))
 * provider/aws: Add retry to the `aws_ssm_document` delete func ([#9188](https://github.com/hashicorp/terraform/issues/9188))
 * provider/aws: Fix issue updating `search_string` in aws_cloudwatch_metric_alarm ([#9230](https://github.com/hashicorp/terraform/issues/9230))
 * provider/aws: Update EFS resource to read performance mode and creation_token ([#9234](https://github.com/hashicorp/terraform/issues/9234))
 * provider/azurerm: fix resource ID parsing for subscriptions resources ([#9163](https://github.com/hashicorp/terraform/issues/9163))
 * provider/librato: Mandatory name and conditions attributes weren't being sent on Update unless changed ([#8984](https://github.com/hashicorp/terraform/issues/8984))
 * provisioner/chef: Fix an error with parsing certain `vault_json` content ([#9114](https://github.com/hashicorp/terraform/issues/9114))
 * provisioner/chef: Change to order in which to cleanup the user key so this is done before the Chef run starts ([#9114](https://github.com/hashicorp/terraform/issues/9114))

## 0.7.4 (September 19, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:
 * In previous releases, the `private_key` field in the connection provisioner
   inadvertently accepted a path argument and would read the file contents.
   This functionality has been removed in this release ([#8577](https://github.com/hashicorp/terraform/issues/8577)), and the documented
   method of using the `file()` interpolation function should be used to load
   the key from a file.

FEATURES:
 * **New Resource:** `aws_codecommit_trigger` ([#8751](https://github.com/hashicorp/terraform/issues/8751))
 * **New Resource:** `aws_default_security_group` ([#8861](https://github.com/hashicorp/terraform/issues/8861))
 * **New Remote State Backend:** `manta` ([#8830](https://github.com/hashicorp/terraform/issues/8830))

IMPROVEMENTS:
 * provider/aws: Support 'publish' attribute in `lambda_function` ([#8653](https://github.com/hashicorp/terraform/issues/8653))
 * provider/aws: Add `reader_endpoint` RDS Clusters ([#8884](https://github.com/hashicorp/terraform/issues/8884))
 * provider/aws: Export AWS ELB service account ARN ([#8700](https://github.com/hashicorp/terraform/issues/8700))
 * provider/aws: Allow `aws_alb` to have the name auto-generated ([#8673](https://github.com/hashicorp/terraform/issues/8673))
 * provider/aws: Expose `arn_suffix` on `aws_alb` ([#8833](https://github.com/hashicorp/terraform/issues/8833))
 * provider/aws: Add JSON validation to the `aws_cloudformation_stack` resource ([#8896](https://github.com/hashicorp/terraform/issues/8896))
 * provider/aws: Add JSON validation to the `aws_glacier_vault` resource ([#8899](https://github.com/hashicorp/terraform/issues/8899))
 * provider/azurerm: support Diagnostics Profile ([#8277](https://github.com/hashicorp/terraform/issues/8277))
 * provider/google: Resources depending on the `network` attribute can now reference the network by `self_link` or `name` ([#8639](https://github.com/hashicorp/terraform/issues/8639))
 * provider/postgresql: The standard environment variables PGHOST, PGUSER, PGPASSWORD and PGSSLMODE are now supported for provider configuration ([#8666](https://github.com/hashicorp/terraform/issues/8666))
 * helper/resource: Add timeout duration to timeout error message ([#8773](https://github.com/hashicorp/terraform/issues/8773))
 * provisioner/chef: Support recreating Chef clients by setting `recreate_client=true` ([#8577](https://github.com/hashicorp/terraform/issues/8577))
 * provisioner/chef: Support encrypting existing Chef-Vaults for newly created clients ([#8577](https://github.com/hashicorp/terraform/issues/8577))

BUG FIXES:
 * core: Fix regression when loading variables from json ([#8820](https://github.com/hashicorp/terraform/issues/8820))
 * provider/aws: Prevent crash creating an `aws_sns_topic` with an empty policy ([#8834](https://github.com/hashicorp/terraform/issues/8834))
 * provider/aws: Bump `aws_elasticsearch_domain` timeout values ([#672](https://github.com/hashicorp/terraform/issues/672))
 * provider/aws: `aws_nat_gateways` will now recreate on `failed` state ([#8689](https://github.com/hashicorp/terraform/issues/8689))
 * provider/aws: Prevent crash on account ID validation ([#8731](https://github.com/hashicorp/terraform/issues/8731))
 * provider/aws: `aws_db_instance` unexpected state when configurating enhanced monitoring ([#8707](https://github.com/hashicorp/terraform/issues/8707))
 * provider/aws: Remove region condition from `aws_codecommit_repository` ([#8778](https://github.com/hashicorp/terraform/issues/8778))
 * provider/aws: Support Policy DiffSuppression in `aws_kms_key` policy ([#8675](https://github.com/hashicorp/terraform/issues/8675))
 * provider/aws: Fix issue updating Elastic Beanstalk Environment variables ([#8848](https://github.com/hashicorp/terraform/issues/8848))
 * provider/scaleway: Fix `security_group_rule` identification ([#8661](https://github.com/hashicorp/terraform/issues/8661))
 * provider/cloudstack: Fix renaming a VPC with the `cloudstack_vpc` resource ([#8784](https://github.com/hashicorp/terraform/issues/8784))

## 0.7.3 (September 5, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:
 * Terraform now validates the uniqueness of variable and output names in your configurations. In prior versions certain ways of duplicating variable names would work. This is now a configuration error (and should've always been). If you get an error running Terraform you may need to remove the duplicates. Done right, this should not affect the behavior of Terraform.
 * The internal structure of `.terraform/modules` changed slightly. For configurations with modules, you'll need to run `terraform get` again.

FEATURES:
 * **New Provider:** `rabbitmq` ([#7694](https://github.com/hashicorp/terraform/issues/7694))
 * **New Data Source:** `aws_cloudformation_stack` ([#8640](https://github.com/hashicorp/terraform/issues/8640))
 * **New Resource:** `aws_cloudwatch_log_stream` ([#8626](https://github.com/hashicorp/terraform/issues/8626))
 * **New Resource:** `aws_default_route_table` ([#8323](https://github.com/hashicorp/terraform/issues/8323))
 * **New Resource:** `aws_spot_datafeed_subscription` ([#8640](https://github.com/hashicorp/terraform/issues/8640))
 * **New Resource:** `aws_s3_bucket_policy` ([#8615](https://github.com/hashicorp/terraform/issues/8615))
 * **New Resource:** `aws_sns_topic_policy` ([#8654](https://github.com/hashicorp/terraform/issues/8654))
 * **New Resource:** `aws_sqs_queue_policy` ([#8657](https://github.com/hashicorp/terraform/issues/8657))
 * **New Resource:** `aws_ssm_association` ([#8376](https://github.com/hashicorp/terraform/issues/8376))
 * **New Resource:** `cloudstack_affinity_group` ([#8360](https://github.com/hashicorp/terraform/issues/8360))
 * **New Resource:** `librato_alert` ([#8170](https://github.com/hashicorp/terraform/issues/8170))
 * **New Resource:** `librato_service` ([#8170](https://github.com/hashicorp/terraform/issues/8170))
 * **New Remote State Backend:** `local` ([#8647](https://github.com/hashicorp/terraform/issues/8647))
 * Data source blocks can now have a count associated with them ([#8635](https://github.com/hashicorp/terraform/issues/8635))
 * The count of a resource can now be referenced for interpolations: `self.count` and `type.name.count` work ([#8581](https://github.com/hashicorp/terraform/issues/8581))
 * Provisioners now support connection using IPv6 in addition to IPv4 ([#6616](https://github.com/hashicorp/terraform/issues/6616))

IMPROVEMENTS:
 * core: Add wildcard (match all) support to `ignore_changes` ([#8599](https://github.com/hashicorp/terraform/issues/8599))
 * core: HTTP module sources can now use netrc files for auth
 * core: Show last resource state in a timeout error message ([#8510](https://github.com/hashicorp/terraform/issues/8510))
 * helper/schema: Add diff suppression callback ([#8585](https://github.com/hashicorp/terraform/issues/8585))
 * provider/aws: API Gateway Custom Authorizer ([#8535](https://github.com/hashicorp/terraform/issues/8535))
 * provider/aws: Add MemoryReservation To `aws_ecs_container_definition` data source ([#8437](https://github.com/hashicorp/terraform/issues/8437))
 * provider/aws: Add ability Enable/Disable For ELB Access logs ([#8438](https://github.com/hashicorp/terraform/issues/8438))
 * provider/aws: Add support for assuming a role prior to performing API operations ([#8638](https://github.com/hashicorp/terraform/issues/8638))
 * provider/aws: Export `arn` of `aws_autoscaling_group` ([#8503](https://github.com/hashicorp/terraform/issues/8503))
 * provider/aws: More robust handling of Lambda function archives hosted on S3 ([#6860](https://github.com/hashicorp/terraform/issues/6860))
 * provider/aws: Spurious diffs of `aws_s3_bucket` policy attributes due to JSON field ordering are reduced ([#8615](https://github.com/hashicorp/terraform/issues/8615))
 * provider/aws: `name_regex` attribute for local post-filtering of `aws_ami` data source results ([#8403](https://github.com/hashicorp/terraform/issues/8403))
 * provider/aws: Support for lifecycle hooks at ASG creation ([#5620](https://github.com/hashicorp/terraform/issues/5620))
 * provider/consul: Make provider settings truly optional ([#8551](https://github.com/hashicorp/terraform/issues/8551))
 * provider/statuscake: Add support for contact-group id in statuscake test ([#8417](https://github.com/hashicorp/terraform/issues/8417))

BUG FIXES:
 * core: Changing a module source from file to VCS no longer errors ([#8398](https://github.com/hashicorp/terraform/issues/8398))
 * core: Configuration is now validated prior to input, fixing an obscure parse error when attempting to interpolate a count ([#8591](https://github.com/hashicorp/terraform/issues/8591))
 * core: JSON configuration with resources with a single key parse properly ([#8485](https://github.com/hashicorp/terraform/issues/8485))
 * core: States with duplicate modules are detected and an error is shown ([#8463](https://github.com/hashicorp/terraform/issues/8463))
 * core: Validate uniqueness of variables/outputs in a module ([#8482](https://github.com/hashicorp/terraform/issues/8482))
 * core: `-var` flag inputs starting with `/` work
 * core: `-var` flag inputs starting with a number work and was fixed in such a way that this should overall be a lot more resilient to inputs ([#8044](https://github.com/hashicorp/terraform/issues/8044))
 * provider/aws: Add AWS error message to retry APIGateway account update ([#8533](https://github.com/hashicorp/terraform/issues/8533))
 * provider/aws: Do not set empty string to state for `aws_vpn_gateway` availability zone ([#8645](https://github.com/hashicorp/terraform/issues/8645))
 * provider/aws: Fix. Adjust create and destroy timeout in aws_vpn_gateway_attachment. ([#8636](https://github.com/hashicorp/terraform/issues/8636))
 * provider/aws: Handle missing EFS mount target in `aws_efs_mount_target` ([#8529](https://github.com/hashicorp/terraform/issues/8529))
 * provider/aws: If an `aws_security_group` was used in Lambda function it may have prevented you from destroying such SG due to dangling ENIs created by Lambda service. These ENIs are now automatically cleaned up prior to SG deletion ([#8033](https://github.com/hashicorp/terraform/issues/8033))
 * provider/aws: Increase `aws_route_table` timeouts from 1 min to 2 mins ([#8465](https://github.com/hashicorp/terraform/issues/8465))
 * provider/aws: Increase aws_rds_cluster timeout to 40 minutes ([#8623](https://github.com/hashicorp/terraform/issues/8623))
 * provider/aws: Refresh `aws_route` from state if `aws_route_table` not found ([#8443](https://github.com/hashicorp/terraform/issues/8443))
 * provider/aws: Remove `aws_elasticsearch_domain` from state if it doesn't exist ([#8643](https://github.com/hashicorp/terraform/issues/8643))
 * provider/aws: Remove unsafe ptr dereferencing from ECS/ECR ([#8514](https://github.com/hashicorp/terraform/issues/8514))
 * provider/aws: Set `apply_method` to state in `aws_db_parameter_group` ([#8603](https://github.com/hashicorp/terraform/issues/8603))
 * provider/aws: Stop `aws_instance` `source_dest_check` triggering an API call on each terraform run ([#8450](https://github.com/hashicorp/terraform/issues/8450))
 * provider/aws: Wait for `aws_route_53_record` to be in-sync after a delete ([#8646](https://github.com/hashicorp/terraform/issues/8646))
 * provider/aws: `aws_volume_attachment` detachment errors are caught ([#8479](https://github.com/hashicorp/terraform/issues/8479))
 * provider/aws: adds resource retry to `aws_spot_instance_request` ([#8516](https://github.com/hashicorp/terraform/issues/8516))
 * provider/aws: Add validation of Health Check target to aws_elb. ([#8578](https://github.com/hashicorp/terraform/issues/8578))
 * provider/aws: Skip detaching when aws_internet_gateway not found ([#8454](https://github.com/hashicorp/terraform/issues/8454))
 * provider/aws: Handle all kinds of CloudFormation stack failures ([#5606](https://github.com/hashicorp/terraform/issues/5606))
 * provider/azurerm: Reordering the checks after an Azure API Get ([#8607](https://github.com/hashicorp/terraform/issues/8607))
 * provider/chef: Fix "invalid header" errors that could occur ([#8382](https://github.com/hashicorp/terraform/issues/8382))
 * provider/github: Remove unsafe ptr dereferencing ([#8512](https://github.com/hashicorp/terraform/issues/8512))
 * provider/librato: Refresh space from state when not found ([#8596](https://github.com/hashicorp/terraform/issues/8596))
 * provider/mysql: Fix breakage in parsing MySQL version string ([#8571](https://github.com/hashicorp/terraform/issues/8571))
 * provider/template: `template_file` vars can be floating point ([#8590](https://github.com/hashicorp/terraform/issues/8590))
 * provider/triton: Fix bug where the ID of a `triton_key` was used prior to being set ([#8563](https://github.com/hashicorp/terraform/issues/8563))

## 0.7.2 (August 25, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:
 * provider/openstack: changes were made to how volumes attached to instances are detected. If you attached a volume to an instance out of band to Terraform, it will be detached upon the next apply. You can resolve this by adding a `volume` entry for the attached volume.
 * provider/aws: `aws_spot_fleet_request` has changed the `associate_public_ip_address` default from `true` to `false`

FEATURES:
 * **New Resource:** `aws_api_gateway_base_path_mapping` ([#8353](https://github.com/hashicorp/terraform/issues/8353))
 * **New Resource:** `aws_api_gateway_domain_name` ([#8353](https://github.com/hashicorp/terraform/issues/8353))
 * **New Resource:** `aws_ssm_document` ([#8460](https://github.com/hashicorp/terraform/issues/8460))

IMPROVEMENTS:
 * core: Names generated with a unique prefix are now sortable based on age ([#8249](https://github.com/hashicorp/terraform/issues/8249))
 * provider/aws: Add Primary Endpoint Address attribute for `aws_elasticache_replication_group` ([#8385](https://github.com/hashicorp/terraform/issues/8385))
 * provider/aws: Add support for `network_mode` to `aws_ecs_task_definition` ([#8391](https://github.com/hashicorp/terraform/issues/8391))
 * provider/aws: Add support for LB target group to ECS service ([#8190](https://github.com/hashicorp/terraform/issues/8190))
 * provider/aws: Support Tags for `aws_alb` and `aws_alb_target_group` resources ([#8422](https://github.com/hashicorp/terraform/issues/8422))
 * provider/aws: Support `snapshot_name` for ElastiCache Cluster and Replication Groups ([#8419](https://github.com/hashicorp/terraform/issues/8419))
 * provider/aws: Add support to `aws_redshift_cluster` for restoring from snapshot ([#8414](https://github.com/hashicorp/terraform/issues/8414))
 * provider/aws: Add validation for master_password in `aws_redshift_cluster` ([#8434](https://github.com/hashicorp/terraform/issues/8434))
 * provider/openstack: Add `allowed_address_pairs` to `openstack_networking_port_v2` ([#8257](https://github.com/hashicorp/terraform/issues/8257))

BUG FIXES:
 * core: fix crash case when malformed JSON given ([#8295](https://github.com/hashicorp/terraform/issues/8295))
 * core: when asking for input, spaces are allowed ([#8394](https://github.com/hashicorp/terraform/issues/8394))
 * core: module sources with URL encodings in the local file path won't error ([#8418](https://github.com/hashicorp/terraform/issues/8418))
 * command/apply: prefix destroying resources with module path ([#8396](https://github.com/hashicorp/terraform/issues/8396))
 * command/import: can import into specific indexes ([#8335](https://github.com/hashicorp/terraform/issues/8335))
 * command/push: -upload-modules=false works ([#8456](https://github.com/hashicorp/terraform/issues/8456))
 * command/state mv: nested modules can be moved ([#8304](https://github.com/hashicorp/terraform/issues/8304))
 * command/state mv: resources with a count > 1 can be moved ([#8304](https://github.com/hashicorp/terraform/issues/8304))
 * provider/aws: Refresh `aws_lambda_event_source_mapping` from state when NotFound ([#8378](https://github.com/hashicorp/terraform/issues/8378))
 * provider/aws: `aws_elasticache_replication_group_id` validation change ([#8381](https://github.com/hashicorp/terraform/issues/8381))
 * provider/aws: Fix possible crash if using duplicate Route53 records ([#8399](https://github.com/hashicorp/terraform/issues/8399))
 * provider/aws: Refresh `aws_autoscaling_policy` from state on 404 ([#8430](https://github.com/hashicorp/terraform/issues/8430))
 * provider/aws: Fix crash with VPC Peering connection accept/requests ([#8432](https://github.com/hashicorp/terraform/issues/8432))
 * provider/aws: AWS SpotFleet Requests now works with Subnets and AZs ([#8320](https://github.com/hashicorp/terraform/issues/8320))
 * provider/aws: Refresh `aws_cloudwatch_event_target` from state on `ResourceNotFoundException` ([#8442](https://github.com/hashicorp/terraform/issues/8442))
 * provider/aws: Validate `aws_iam_policy_attachment` Name parameter to stop being empty ([#8441](https://github.com/hashicorp/terraform/issues/8441))
 * provider/aws: Fix segmentation fault in `aws_api_gateway_base_path_mapping` resource ([#8466](https://github.com/hashicorp/terraform/issues/8466))
 * provider/google: fix crash regression from Terraform 0.7.1 on `google_compute_firewall` resource ([#8390](https://github.com/hashicorp/terraform/issues/8390))
 * provider/openstack: Volume Attachment and Detachment Fixes ([#8172](https://github.com/hashicorp/terraform/issues/8172))

## 0.7.1 (August 19, 2016)

FEATURES:
 * **New Command:** `terraform state rm` ([#8200](https://github.com/hashicorp/terraform/issues/8200))
 * **New Provider:** `archive` ([#7322](https://github.com/hashicorp/terraform/issues/7322))
 * **New Resource:** `aws_alb` ([#8254](https://github.com/hashicorp/terraform/issues/8254))
 * **New Resource:** `aws_alb_listener` ([#8269](https://github.com/hashicorp/terraform/issues/8269))
 * **New Resource:** `aws_alb_target_group` ([#8254](https://github.com/hashicorp/terraform/issues/8254))
 * **New Resource:** `aws_alb_target_group_attachment` ([#8254](https://github.com/hashicorp/terraform/issues/8254))
 * **New Resource:** `aws_alb_target_group_rule` ([#8321](https://github.com/hashicorp/terraform/issues/8321))
 * **New Resource:** `aws_vpn_gateway_attachment` ([#7870](https://github.com/hashicorp/terraform/issues/7870))
 * **New Resource:** `aws_load_balancer_policy` ([#7458](https://github.com/hashicorp/terraform/issues/7458))
 * **New Resource:** `aws_load_balancer_backend_server_policy` ([#7458](https://github.com/hashicorp/terraform/issues/7458))
 * **New Resource:** `aws_load_balancer_listener_policy` ([#7458](https://github.com/hashicorp/terraform/issues/7458))
 * **New Resource:** `aws_lb_ssl_negotiation_policy` ([#8084](https://github.com/hashicorp/terraform/issues/8084))
 * **New Resource:** `aws_elasticache_replication_groups` ([#8275](https://github.com/hashicorp/terraform/issues/8275))
 * **New Resource:** `azurerm_virtual_network_peering` ([#8168](https://github.com/hashicorp/terraform/issues/8168))
 * **New Resource:** `azurerm_servicebus_namespace` ([#8195](https://github.com/hashicorp/terraform/issues/8195))
 * **New Resource:** `google_compute_image` ([#7960](https://github.com/hashicorp/terraform/issues/7960))
 * **New Resource:** `packet_volume` ([#8142](https://github.com/hashicorp/terraform/issues/8142))
 * **New Resource:** `consul_prepared_query` ([#7474](https://github.com/hashicorp/terraform/issues/7474))
 * **New Data Source:** `aws_ip_ranges` ([#7984](https://github.com/hashicorp/terraform/issues/7984))
 * **New Data Source:** `fastly_ip_ranges` ([#7984](https://github.com/hashicorp/terraform/issues/7984))
 * **New Data Source:** `aws_caller_identity` ([#8206](https://github.com/hashicorp/terraform/issues/8206))
 * **New Data Source:** `aws_elb_service_account` ([#8221](https://github.com/hashicorp/terraform/issues/8221))
 * **New Data Source:** `aws_redshift_service_account` ([#8224](https://github.com/hashicorp/terraform/issues/8224))

IMPROVEMENTS
 * provider/archive support folders in output_path ([#8278](https://github.com/hashicorp/terraform/issues/8278))
 * provider/aws: Introduce `aws_elasticsearch_domain` `elasticsearch_version` field (to specify ES version) ([#7860](https://github.com/hashicorp/terraform/issues/7860))
 * provider/aws: Add support for TargetGroups (`aws_alb_target_groups`) to `aws_autoscaling_group` [8327]
 * provider/aws: CloudWatch Metrics are now supported for `aws_route53_health_check` resources ([#8319](https://github.com/hashicorp/terraform/issues/8319))
 * provider/aws: Query all pages of group membership ([#6726](https://github.com/hashicorp/terraform/issues/6726))
 * provider/aws: Query all pages of IAM Policy attachments ([#7779](https://github.com/hashicorp/terraform/issues/7779))
 * provider/aws: Change the way ARNs are built ([#7151](https://github.com/hashicorp/terraform/issues/7151))
 * provider/aws: Add support for Elasticsearch destination to firehose delivery streams ([#7839](https://github.com/hashicorp/terraform/issues/7839))
 * provider/aws: Retry AttachInternetGateway and increase timeout on `aws_internet_gateway` ([#7891](https://github.com/hashicorp/terraform/issues/7891))
 * provider/aws: Add support for Enhanced monitoring to `aws_rds_cluster_instance` ([#8038](https://github.com/hashicorp/terraform/issues/8038))
 * provider/aws: Add ability to set Requests Payer in `aws_s3_bucket` ([#8065](https://github.com/hashicorp/terraform/issues/8065))
 * provider/aws: Add ability to set canned ACL in `aws_s3_bucket_object` ([#8091](https://github.com/hashicorp/terraform/issues/8091))
 * provider/aws: Allow skipping credentials validation, requesting Account ID and/or metadata API check ([#7874](https://github.com/hashicorp/terraform/issues/7874))
 * provider/aws: API gateway request/response parameters can now be specified as map, original `*_in_json` parameters  deprecated ([#7794](https://github.com/hashicorp/terraform/issues/7794))
 * provider/aws: Add support for `promotion_tier` to `aws_rds_cluster_instance` ([#8087](https://github.com/hashicorp/terraform/issues/8087))
 * provider/aws: Allow specifying custom S3 endpoint and enforcing S3 path style URLs via new provider options ([#7871](https://github.com/hashicorp/terraform/issues/7871))
 * provider/aws: Add ability to set Storage Class in `aws_s3_bucket_object` ([#8174](https://github.com/hashicorp/terraform/issues/8174))
 * provider/aws: Treat `aws_lambda_function` w/ empty `subnet_ids` & `security_groups_ids` in `vpc_config` as VPC-disabled function ([#6191](https://github.com/hashicorp/terraform/issues/6191))
 * provider/aws: Allow `source_ids` in `aws_db_event_subscription` to be Updatable ([#7892](https://github.com/hashicorp/terraform/issues/7892))
 * provider/aws: Make `aws_efs_mount_target` creation fail for 2+ targets per AZ ([#8205](https://github.com/hashicorp/terraform/issues/8205))
 * provider/aws: Add `force_destroy` option to `aws_route53_zone` ([#8239](https://github.com/hashicorp/terraform/issues/8239))
 * provider/aws: Support import of `aws_s3_bucket` ([#8262](https://github.com/hashicorp/terraform/issues/8262))
 * provider/aws: Increase timeout for retrying creation of IAM role ([#7733](https://github.com/hashicorp/terraform/issues/7733))
 * provider/aws: Add ability to set peering options in aws_vpc_peering_connection. ([#8310](https://github.com/hashicorp/terraform/issues/8310))
 * provider/azure: add custom_data argument for azure_instance resource ([#8158](https://github.com/hashicorp/terraform/issues/8158))
 * provider/azurerm: Adds support for uploading blobs to azure storage from local source ([#7994](https://github.com/hashicorp/terraform/issues/7994))
 * provider/azurerm: Storage blob contents can be copied from an existing blob ([#8126](https://github.com/hashicorp/terraform/issues/8126))
 * provider/datadog: Allow `tags` to be configured for monitor resources. ([#8284](https://github.com/hashicorp/terraform/issues/8284))
 * provider/google: allows atomic Cloud DNS record changes ([#6575](https://github.com/hashicorp/terraform/issues/6575))
 * provider/google: Move URLMap hosts to TypeSet from TypeList ([#7472](https://github.com/hashicorp/terraform/issues/7472))
 * provider/google: Support static private IP addresses in `resource_compute_instance` ([#6310](https://github.com/hashicorp/terraform/issues/6310))
 * provider/google: Add support for using a GCP Image Family ([#8083](https://github.com/hashicorp/terraform/issues/8083))
 * provider/openstack: Support updating the External Gateway assigned to a Neutron router ([#8070](https://github.com/hashicorp/terraform/issues/8070))
 * provider/openstack: Support for `value_specs` param on `openstack_networking_network_v2` ([#8155](https://github.com/hashicorp/terraform/issues/8155))
 * provider/openstack: Add `value_specs` param on `openstack_networking_subnet_v2` ([#8181](https://github.com/hashicorp/terraform/issues/8181))
 * provider/vsphere: Improved SCSI controller handling in `vsphere_virtual_machine` ([#7908](https://github.com/hashicorp/terraform/issues/7908))
 * provider/vsphere: Adding disk type of `Thick Lazy` to `vsphere_virtual_disk` and `vsphere_virtual_machine` ([#7916](https://github.com/hashicorp/terraform/issues/7916))
 * provider/vsphere: Standardizing datastore references to use builtin Path func ([#8075](https://github.com/hashicorp/terraform/issues/8075))
 * provider/consul: add tls config support to consul provider ([#7015](https://github.com/hashicorp/terraform/issues/7015))
 * remote/consul: Support setting datacenter when using consul remote state ([#8102](https://github.com/hashicorp/terraform/issues/8102))
 * provider/google: Support import of `google_compute_instance_template` ([#8147](https://github.com/hashicorp/terraform/issues/8147)), `google_compute_firewall` ([#8236](https://github.com/hashicorp/terraform/issues/8236)), `google_compute_target_pool` ([#8133](https://github.com/hashicorp/terraform/issues/8133)), `google_compute_fowarding_rule` ([#8122](https://github.com/hashicorp/terraform/issues/8122)), `google_compute_http_health_check` ([#8121](https://github.com/hashicorp/terraform/issues/8121)), `google_compute_autoscaler` ([#8115](https://github.com/hashicorp/terraform/issues/8115))

BUG FIXES:
 * core: Fix issue preventing `taint` from working with resources that had no other attributes in their diff ([#8167](https://github.com/hashicorp/terraform/issues/8167))
 * core: CLI will only run exact match commands ([#7983](https://github.com/hashicorp/terraform/issues/7983))
 * core: Fix panic when resources ends up null in state file ([#8120](https://github.com/hashicorp/terraform/issues/8120))
 * core: Fix panic when validating a count with a unprefixed variable ([#8243](https://github.com/hashicorp/terraform/issues/8243))
 * core: Divide by zero in interpolations no longer panics ([#7701](https://github.com/hashicorp/terraform/issues/7701))
 * core: Fix panic on some invalid interpolation syntax ([#5672](https://github.com/hashicorp/terraform/issues/5672))
 * provider/aws: guard against missing image_digest in `aws_ecs_task_definition` ([#7966](https://github.com/hashicorp/terraform/issues/7966))
 * provider/aws: `aws_cloudformation_stack` now respects `timeout_in_minutes` field when waiting for CF API to finish an update operation ([#7997](https://github.com/hashicorp/terraform/issues/7997))
 * provider/aws: Prevent errors when `aws_s3_bucket` `acceleration_status` is not available in a given region ([#7999](https://github.com/hashicorp/terraform/issues/7999))
 * provider/aws: Add state filter to `aws_availability_zone`s data source ([#7965](https://github.com/hashicorp/terraform/issues/7965))
 * provider/aws: Handle lack of snapshot ID for a volume in `ami_copy` ([#7995](https://github.com/hashicorp/terraform/issues/7995))
 * provider/aws: Retry association of IAM Role & instance profile ([#7938](https://github.com/hashicorp/terraform/issues/7938))
 * provider/aws: Fix `aws_s3_bucket` resource `redirect_all_requests_to` action ([#7883](https://github.com/hashicorp/terraform/issues/7883))
 * provider/aws: Fix issue updating ElasticBeanstalk Environment Settings ([#7777](https://github.com/hashicorp/terraform/issues/7777))
 * provider/aws: `aws_rds_cluster` creation timeout bumped to 40 minutes ([#8052](https://github.com/hashicorp/terraform/issues/8052))
 * provider/aws: Update ElasticTranscoder to allow empty notifications, removing notifications, etc ([#8207](https://github.com/hashicorp/terraform/issues/8207))
 * provider/aws: Fix line ending errors/diffs with IAM Server Certs ([#8074](https://github.com/hashicorp/terraform/issues/8074))
 * provider/aws: Fixing IAM data source policy generation to prevent spurious diffs ([#6956](https://github.com/hashicorp/terraform/issues/6956))
 * provider/aws: Correct how CORS rules are handled in `aws_s3_bucket` ([#8096](https://github.com/hashicorp/terraform/issues/8096))
 * provider/aws: allow numeric characters in RedshiftClusterDbName ([#8178](https://github.com/hashicorp/terraform/issues/8178))
 * provider/aws: `aws_security_group` now creates tags as early as possible in the process ([#7849](https://github.com/hashicorp/terraform/issues/7849))
 * provider/aws: Defensively code around `db_security_group` ingress rules ([#7893](https://github.com/hashicorp/terraform/issues/7893))
 * provider/aws: `aws_spot_fleet_request` throws panic on missing subnet_id or availability_zone ([#8217](https://github.com/hashicorp/terraform/issues/8217))
 * provider/aws: Terraform fails during Redshift delete if FinalSnapshot is being taken. ([#8270](https://github.com/hashicorp/terraform/issues/8270))
 * provider/azurerm: `azurerm_storage_account` will interrupt for Ctrl-C ([#8215](https://github.com/hashicorp/terraform/issues/8215))
 * provider/azurerm: Public IP - Setting idle timeout value caused panic. #8283
 * provider/digitalocean: trim whitespace from ssh key ([#8173](https://github.com/hashicorp/terraform/issues/8173))
 * provider/digitalocean: Enforce Lowercase on IPV6 Addresses ([#7652](https://github.com/hashicorp/terraform/issues/7652))
 * provider/google: Use resource specific project when making queries/changes ([#7029](https://github.com/hashicorp/terraform/issues/7029))
 * provider/google: Fix read for the backend service resource ([#7476](https://github.com/hashicorp/terraform/issues/7476))
 * provider/mysql: `mysql_user` works with MySQL versions before 5.7.6 ([#8251](https://github.com/hashicorp/terraform/issues/8251))
 * provider/openstack: Fix typo in OpenStack LBaaSv2 pool resource ([#8179](https://github.com/hashicorp/terraform/issues/8179))
 * provider/vSphere: Fix for IPv6 only environment creation ([#7643](https://github.com/hashicorp/terraform/issues/7643))
 * provider/google: Correct update process for authorized networks in `google_sql_database_instance` ([#8290](https://github.com/hashicorp/terraform/issues/8290))

## 0.7.0 (August 2, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

  * Terraform Core
   * Terraform's built-in plugins are now distributed as part of the main Terraform binary, and use the go-plugin framework. Overrides are still available using separate binaries, but will need recompiling against Terraform 0.7.
   * The `terraform plan` command no longer persists state. This makes the command much safer to run, since it is now side-effect free. The `refresh` and `apply` commands still persist state to local and remote storage. Any automation that assumes that `terraform plan` persists state will need to be reworked to explicitly call `terraform refresh` to get the equivalent side-effect. (The `terraform plan` command no longer has the `-state-out` or `-backup` flags due to this change.)
   * The `concat()` interpolation function can no longer be used to join strings.
   * Quotation marks may no longer be escaped in HIL expressions ([#7201](https://github.com/hashicorp/terraform/issues/7201))
   * Lists materialized using splat syntax, for example `aws_instance.foo.*.id` are now ordered by the count index rather than lexographically sorted. If this produces a large number of undesirable differences, you can use the new `sort()` interpolation function to produce the previous behaviour.
   * You now access the values of maps using the syntax `var.map["key"]` or the `lookup` function instead of `var.map.key`.
   * Outputs on `terraform_remote_state` resources are now top level attributes rather than inside the `output` map. In order to access outputs, use the syntax: `terraform_remote_state.name.outputname`. Currently outputs cannot be named `config` or `backend`.
  * AWS Provider
   * `aws_elb` now defaults `cross_zone_load_balancing` to `true`
   * `aws_instance`: EC2 Classic users may continue to use `security_groups` to reference Security Groups by their `name`. Users who are managing Instances inside VPCs will need to use `vpc_security_group_ids` instead, and reference the security groups by their `id`. Ref https://github.com/hashicorp/terraform/issues/6416#issuecomment-219145065
   * `aws_kinesis_firehose_delivery_stream`: AWS Kinesis Firehose has been refactored to support Redshift as a destination in addition to S3. As a result, the configuration has changed and users will need to update their configuration to match the new `s3_configuration` block. Checkout the documentaiton on [AWS Kinesis Firehose](http://localhost:4567/docs/providers/aws/r/kinesis_firehose_delivery_stream.html) for more information ([#7375](https://github.com/hashicorp/terraform/issues/7375))
   * `aws_route53_record`: `latency_routing_policy`, `geolocation_routing_policy`, and `failover_routing_policy` block options have been added. With these additions we’ve renamed the `weight` attribute to `weighted_routing_policy`, and it has changed from a string to a block to match the others. Please see the updated documentation on using `weighted_routing_policy`:  https://www.terraform.io/docs/providers/aws/r/route53_record.html . ([#6954](https://github.com/hashicorp/terraform/issues/6954))
   * `aws_db_instance` now defaults `publicly_accessible` to false
  * Microsoft Azure Provider
   * In documentation, the "Azure (Resource Manager)" provider has been renamed to the "Microsoft Azure" provider.
   * `azurerm_dns_cname_record` now accepts a single record rather than a list of records
   * `azurerm_virtual_machine` computer_name now Required
  * Openstack Provider
   * `openstack_networking_subnet_v2` now defaults to turning DHCP on.
   * `openstack_fw_policy_v1` now correctly applies rules in the order they are specified. Upon the next apply, current rules might be re-ordered.
   * The `member` attribute of `openstack_lb_pool_v1` has been deprecated. Please ue the new `openstack_lb_member_v1` resource.
  * Docker Provider
   * `keep_updated` parameter removed from `docker_image` - This parameter never did what it was supposed to do.  See relevant docs, specifically `pull_trigger` & new `docker_registry_image` data source to understand how to keep your `docker_image` updated.
  * Atlas Provider
   * `atlas_artifact` resource has be deprecated. Please use the new `atlas_artifact` Data Source.
  * CloudStack Provider
   * All deprecated parameters are removed from all `CloudStack` resources

FEATURES:

 * **Data sources** are a new kind of primitive in Terraform. Attributes for data sources are refreshed and available during the planning stage. ([#6598](https://github.com/hashicorp/terraform/issues/6598))
 * **Lists and maps** can now be used as first class types for variables and may also be passed between modules. ([#6322](https://github.com/hashicorp/terraform/issues/6322))
 * **State management CLI commands** provide a variety of state manipulation functions for advanced use cases. This should be used where possible instead of manually modifying state files. ([#5811](https://github.com/hashicorp/terraform/issues/5811))
 * **State Import** allows a way to import existing resources into Terraform state for many types of resource. Initial coverage of AWS is quite high, and it is straightforward to add support for new resources.
 * **New Command:** `terraform state` to provide access to a variety of state manipulation functions ([#5811](https://github.com/hashicorp/terraform/issues/5811))
 * **New Option:** `terraform output` now supports the `-json` flag to print a machine-readable representation of outputs ([#7608](https://github.com/hashicorp/terraform/issues/7608))
 * **New Data Source:** `aws_ami` ([#6911](https://github.com/hashicorp/terraform/issues/6911))
 * **New Data Source:** `aws_availability_zones` ([#6805](https://github.com/hashicorp/terraform/issues/6805))
 * **New Data Source:** `aws_iam_policy_document` ([#6881](https://github.com/hashicorp/terraform/issues/6881))
 * **New Data Source:** `aws_s3_bucket_object` ([#6946](https://github.com/hashicorp/terraform/issues/6946))
 * **New Data Source:** `aws_ecs_container_definition` ([#7230](https://github.com/hashicorp/terraform/issues/7230))
 * **New Data Source:** `atlas_artifact` ([#7419](https://github.com/hashicorp/terraform/issues/7419))
 * **New Data Source:** `docker_registry_image` ([#7000](https://github.com/hashicorp/terraform/issues/7000))
 * **New Data Source:** `consul_keys` ([#7678](https://github.com/hashicorp/terraform/issues/7678))
 * **New Interpolation Function:** `sort` ([#7128](https://github.com/hashicorp/terraform/issues/7128))
 * **New Interpolation Function:** `distinct` ([#7174](https://github.com/hashicorp/terraform/issues/7174))
 * **New Interpolation Function:** `list` ([#7528](https://github.com/hashicorp/terraform/issues/7528))
 * **New Interpolation Function:** `map` ([#7832](https://github.com/hashicorp/terraform/issues/7832))
 * **New Provider:** `grafana` ([#6206](https://github.com/hashicorp/terraform/issues/6206))
 * **New Provider:** `logentries` ([#7067](https://github.com/hashicorp/terraform/issues/7067))
 * **New Provider:** `scaleway` ([#7331](https://github.com/hashicorp/terraform/issues/7331))
 * **New Provider:** `random` - allows generation of random values without constantly generating diffs ([#6672](https://github.com/hashicorp/terraform/issues/6672))
 * **New Remote State Provider:** - `gcs` - Google Cloud Storage ([#6814](https://github.com/hashicorp/terraform/issues/6814))
 * **New Remote State Provider:** - `azure` - Microsoft Azure Storage ([#7064](https://github.com/hashicorp/terraform/issues/7064))
 * **New Resource:** `aws_elb_attachment` ([#6879](https://github.com/hashicorp/terraform/issues/6879))
 * **New Resource:** `aws_elastictranscoder_preset` ([#6965](https://github.com/hashicorp/terraform/issues/6965))
 * **New Resource:** `aws_elastictranscoder_pipeline` ([#6965](https://github.com/hashicorp/terraform/issues/6965))
 * **New Resource:** `aws_iam_group_policy_attachment` ([#6858](https://github.com/hashicorp/terraform/issues/6858))
 * **New Resource:** `aws_iam_role_policy_attachment` ([#6858](https://github.com/hashicorp/terraform/issues/6858))
 * **New Resource:** `aws_iam_user_policy_attachment` ([#6858](https://github.com/hashicorp/terraform/issues/6858))
 * **New Resource:** `aws_rds_cluster_parameter_group` ([#5269](https://github.com/hashicorp/terraform/issues/5269))
 * **New Resource:** `aws_spot_fleet_request` ([#7243](https://github.com/hashicorp/terraform/issues/7243))
 * **New Resource:** `aws_ses_active_receipt_rule_set` ([#5387](https://github.com/hashicorp/terraform/issues/5387))
 * **New Resource:** `aws_ses_receipt_filter` ([#5387](https://github.com/hashicorp/terraform/issues/5387))
 * **New Resource:** `aws_ses_receipt_rule` ([#5387](https://github.com/hashicorp/terraform/issues/5387))
 * **New Resource:** `aws_ses_receipt_rule_set` ([#5387](https://github.com/hashicorp/terraform/issues/5387))
 * **New Resource:** `aws_simpledb_domain` ([#7600](https://github.com/hashicorp/terraform/issues/7600))
 * **New Resource:** `aws_opsworks_user_profile` ([#6304](https://github.com/hashicorp/terraform/issues/6304))
 * **New Resource:** `aws_opsworks_permission` ([#6304](https://github.com/hashicorp/terraform/issues/6304))
 * **New Resource:** `aws_ami_launch_permission` ([#7365](https://github.com/hashicorp/terraform/issues/7365))
 * **New Resource:** `aws_appautoscaling_policy` ([#7663](https://github.com/hashicorp/terraform/issues/7663))
 * **New Resource:** `aws_appautoscaling_target` ([#7663](https://github.com/hashicorp/terraform/issues/7663))
 * **New Resource:** `openstack_blockstorage_volume_v2` ([#6693](https://github.com/hashicorp/terraform/issues/6693))
 * **New Resource:** `openstack_lb_loadbalancer_v2` ([#7012](https://github.com/hashicorp/terraform/issues/7012))
 * **New Resource:** `openstack_lb_listener_v2` ([#7012](https://github.com/hashicorp/terraform/issues/7012))
 * **New Resource:** `openstack_lb_pool_v2` ([#7012](https://github.com/hashicorp/terraform/issues/7012))
 * **New Resource:** `openstack_lb_member_v2` ([#7012](https://github.com/hashicorp/terraform/issues/7012))
 * **New Resource:** `openstack_lb_monitor_v2` ([#7012](https://github.com/hashicorp/terraform/issues/7012))
 * **New Resource:** `vsphere_virtual_disk` ([#6273](https://github.com/hashicorp/terraform/issues/6273))
 * **New Resource:** `github_repository_collaborator` ([#6861](https://github.com/hashicorp/terraform/issues/6861))
 * **New Resource:** `datadog_timeboard` ([#6900](https://github.com/hashicorp/terraform/issues/6900))
 * **New Resource:** `digitalocean_tag` ([#7500](https://github.com/hashicorp/terraform/issues/7500))
 * **New Resource:** `digitalocean_volume` ([#7560](https://github.com/hashicorp/terraform/issues/7560))
 * **New Resource:** `consul_agent_service` ([#7508](https://github.com/hashicorp/terraform/issues/7508))
 * **New Resource:** `consul_catalog_entry` ([#7508](https://github.com/hashicorp/terraform/issues/7508))
 * **New Resource:** `consul_node` ([#7508](https://github.com/hashicorp/terraform/issues/7508))
 * **New Resource:** `consul_service` ([#7508](https://github.com/hashicorp/terraform/issues/7508))
 * **New Resource:** `mysql_grant` ([#7656](https://github.com/hashicorp/terraform/issues/7656))
 * **New Resource:** `mysql_user` ([#7656](https://github.com/hashicorp/terraform/issues/7656))
 * **New Resource:** `azurerm_storage_table` ([#7327](https://github.com/hashicorp/terraform/issues/7327))
 * **New Resource:** `azurerm_virtual_machine_scale_set` ([#6711](https://github.com/hashicorp/terraform/issues/6711))
 * **New Resource:** `azurerm_traffic_manager_endpoint` ([#7826](https://github.com/hashicorp/terraform/issues/7826))
 * **New Resource:** `azurerm_traffic_manager_profile` ([#7826](https://github.com/hashicorp/terraform/issues/7826))
 * core: Tainted resources now show up in the plan and respect dependency ordering ([#6600](https://github.com/hashicorp/terraform/issues/6600))
 * core: The `lookup` interpolation function can now have a default fall-back value specified ([#6884](https://github.com/hashicorp/terraform/issues/6884))
 * core: The `terraform plan` command no longer persists state. ([#6811](https://github.com/hashicorp/terraform/issues/6811))

IMPROVEMENTS:

 * core: The `jsonencode` interpolation function now supports encoding lists and maps ([#6749](https://github.com/hashicorp/terraform/issues/6749))
 * core: Add the ability for resource definitions to mark attributes as "sensitive" which will omit them from UI output. ([#6923](https://github.com/hashicorp/terraform/issues/6923))
 * core: Support `.` in map keys ([#7654](https://github.com/hashicorp/terraform/issues/7654))
 * core: Enhance interpolation functions to account for first class maps and lists ([#7832](https://github.com/hashicorp/terraform/issues/7832)) ([#7834](https://github.com/hashicorp/terraform/issues/7834))
 * command: Remove second DefaultDataDirectory const ([#7666](https://github.com/hashicorp/terraform/issues/7666))
 * provider/aws: Add `dns_name` to `aws_efs_mount_target` ([#7428](https://github.com/hashicorp/terraform/issues/7428))
 * provider/aws: Add `force_destroy` to `aws_iam_user` for force-deleting access keys assigned to the user ([#7766](https://github.com/hashicorp/terraform/issues/7766))
 * provider/aws: Add `option_settings` to `aws_db_option_group` ([#6560](https://github.com/hashicorp/terraform/issues/6560))
 * provider/aws: Add more explicit support for Skipping Final Snapshot in RDS Cluster ([#6795](https://github.com/hashicorp/terraform/issues/6795))
 * provider/aws: Add support for S3 Bucket Acceleration ([#6628](https://github.com/hashicorp/terraform/issues/6628))
 * provider/aws: Add support for `kms_key_id` to `aws_db_instance` ([#6651](https://github.com/hashicorp/terraform/issues/6651))
 * provider/aws: Specifying more than one health check on an `aws_elb` fails with an error prior to making an API request ([#7489](https://github.com/hashicorp/terraform/issues/7489))
 * provider/aws: Add support to `aws_redshift_cluster` for `iam_roles` ([#6647](https://github.com/hashicorp/terraform/issues/6647))
 * provider/aws: SQS use raw policy string if compact fails ([#6724](https://github.com/hashicorp/terraform/issues/6724))
 * provider/aws: Set default description to "Managed by Terraform" ([#6104](https://github.com/hashicorp/terraform/issues/6104))
 * provider/aws: Support for Redshift Cluster encryption using a KMS key ([#6712](https://github.com/hashicorp/terraform/issues/6712))
 * provider/aws: Support tags for AWS redshift cluster ([#5356](https://github.com/hashicorp/terraform/issues/5356))
 * provider/aws: Add `iam_arn` to aws_cloudfront_origin_access_identity ([#6955](https://github.com/hashicorp/terraform/issues/6955))
 * provider/aws: Add `cross_zone_load_balancing` on `aws_elb` default to true ([#6897](https://github.com/hashicorp/terraform/issues/6897))
 * provider/aws: Add support for `character_set_name` to `aws_db_instance` ([#4861](https://github.com/hashicorp/terraform/issues/4861))
 * provider/aws: Add support for DB parameter group with RDS Cluster Instances (Aurora) ([#6865](https://github.com/hashicorp/terraform/issues/6865))
 * provider/aws: Add `name_prefix` to `aws_iam_instance_profile` and `aws_iam_role` ([#6939](https://github.com/hashicorp/terraform/issues/6939))
 * provider/aws: Allow authentication & credentials validation for federated IAM Roles and EC2 instance profiles ([#6536](https://github.com/hashicorp/terraform/issues/6536))
 * provider/aws: Rename parameter_group_name to db_cluster_parameter_group_name ([#7083](https://github.com/hashicorp/terraform/issues/7083))
 * provider/aws: Retry RouteTable Route/Assocation creation ([#7156](https://github.com/hashicorp/terraform/issues/7156))
 * provider/aws: `delegation_set_id` conflicts w/ `vpc_id` in `aws_route53_zone` as delegation sets can only be used for public zones ([#7213](https://github.com/hashicorp/terraform/issues/7213))
 * provider/aws: Support Elastic Beanstalk scheduledaction ([#7376](https://github.com/hashicorp/terraform/issues/7376))
 * provider/aws: Add support for NewInstancesProtectedFromScaleIn to `aws_autoscaling_group` ([#6490](https://github.com/hashicorp/terraform/issues/6490))
 * provider/aws: Added support for `snapshot_identifier` parameter in aws_rds_cluster ([#7158](https://github.com/hashicorp/terraform/issues/7158))
 * provider/aws: Add inplace edit/update DB Security Group Rule Ingress ([#7245](https://github.com/hashicorp/terraform/issues/7245))
 * provider/aws: Added support for redshift destination to firehose delivery streams ([#7375](https://github.com/hashicorp/terraform/issues/7375))
 * provider/aws: Allow `aws_redshift_security_group` ingress rules to change ([#5939](https://github.com/hashicorp/terraform/issues/5939))
 * provider/aws: Add support for `encryption` and `kms_key_id` to `aws_ami` ([#7181](https://github.com/hashicorp/terraform/issues/7181))
 * provider/aws: AWS prefix lists to enable security group egress to a VPC Endpoint ([#7511](https://github.com/hashicorp/terraform/issues/7511))
 * provider/aws: Retry creation of IAM role depending on new IAM user ([#7324](https://github.com/hashicorp/terraform/issues/7324))
 * provider/aws: Allow `port` on `aws_db_instance` to be updated ([#7441](https://github.com/hashicorp/terraform/issues/7441))
 * provider/aws: Allow VPC Classic Linking in Autoscaling Launch Configs ([#7470](https://github.com/hashicorp/terraform/issues/7470))
 * provider/aws: Support `task_role_arn` on `aws_ecs_task_definition ([#7653](https://github.com/hashicorp/terraform/issues/7653))
 * provider/aws: Support Tags on `aws_rds_cluster` ([#7695](https://github.com/hashicorp/terraform/issues/7695))
 * provider/aws: Support kms_key_id for `aws_rds_cluster` ([#7662](https://github.com/hashicorp/terraform/issues/7662))
 * provider/aws: Allow setting a `poll_interval` on `aws_elastic_beanstalk_environment` ([#7523](https://github.com/hashicorp/terraform/issues/7523))
 * provider/aws: Add support for Kinesis streams shard-level metrics ([#7684](https://github.com/hashicorp/terraform/issues/7684))
 * provider/aws: Support create / update greater than twenty db parameters in `aws_db_parameter_group` ([#7364](https://github.com/hashicorp/terraform/issues/7364))
 * provider/aws: expose network interface id in `aws_instance` ([#6751](https://github.com/hashicorp/terraform/issues/6751))
 * provider/aws: Adding passthrough behavior for API Gateway integration ([#7801](https://github.com/hashicorp/terraform/issues/7801))
 * provider/aws: Enable Redshift Cluster Logging ([#7813](https://github.com/hashicorp/terraform/issues/7813))
 * provider/aws: Add ability to set Performance Mode in `aws_efs_file_system` ([#7791](https://github.com/hashicorp/terraform/issues/7791))
 * provider/azurerm: Add support for EnableIPForwarding to `azurerm_network_interface` ([#6807](https://github.com/hashicorp/terraform/issues/6807))
 * provider/azurerm: Add support for exporting the `azurerm_storage_account` access keys ([#6742](https://github.com/hashicorp/terraform/issues/6742))
 * provider/azurerm: The Azure SDK now exposes better error messages ([#6976](https://github.com/hashicorp/terraform/issues/6976))
 * provider/azurerm: `azurerm_dns_zone` now returns `name_servers` ([#7434](https://github.com/hashicorp/terraform/issues/7434))
 * provider/azurerm: dump entire Request/Response in autorest Decorator ([#7719](https://github.com/hashicorp/terraform/issues/7719))
 * provider/azurerm: add option to delete VMs Data disks on termination ([#7793](https://github.com/hashicorp/terraform/issues/7793))
 * provider/clc: Add support for hyperscale and bareMetal server types and package installation
 * provider/clc: Fix optional server password ([#6414](https://github.com/hashicorp/terraform/issues/6414))
 * provider/cloudstack: Add support for affinity groups to `cloudstack_instance` ([#6898](https://github.com/hashicorp/terraform/issues/6898))
 * provider/cloudstack: Enable swapping of ACLs without having to rebuild the network tier ([#6741](https://github.com/hashicorp/terraform/issues/6741))
 * provider/cloudstack: Improve ACL swapping ([#7315](https://github.com/hashicorp/terraform/issues/7315))
 * provider/cloudstack: Add project support to `cloudstack_network_acl` and `cloudstack_network_acl_rule` ([#7612](https://github.com/hashicorp/terraform/issues/7612))
 * provider/cloudstack: Add option to set `root_disk_size` to `cloudstack_instance` ([#7070](https://github.com/hashicorp/terraform/issues/7070))
 * provider/cloudstack: Do no longer force a new `cloudstack_instance` resource when updating `user_data` ([#7074](https://github.com/hashicorp/terraform/issues/7074))
 * provider/cloudstack: Add option to set `security_group_names` to `cloudstack_instance` ([#7240](https://github.com/hashicorp/terraform/issues/7240))
 * provider/cloudstack: Add option to set `affinity_group_names` to `cloudstack_instance` ([#7242](https://github.com/hashicorp/terraform/issues/7242))
 * provider/datadog: Add support for 'require full window' and 'locked' ([#6738](https://github.com/hashicorp/terraform/issues/6738))
 * provider/docker: Docker Container DNS Setting Enhancements ([#7392](https://github.com/hashicorp/terraform/issues/7392))
 * provider/docker: Add `destroy_grace_seconds` option to stop container before delete ([#7513](https://github.com/hashicorp/terraform/issues/7513))
 * provider/docker: Add `pull_trigger` option to `docker_image` to trigger pulling layers of a given image ([#7000](https://github.com/hashicorp/terraform/issues/7000))
 * provider/fastly: Add support for Cache Settings ([#6781](https://github.com/hashicorp/terraform/issues/6781))
 * provider/fastly: Add support for Service Request Settings on `fastly_service_v1` resources ([#6622](https://github.com/hashicorp/terraform/issues/6622))
 * provider/fastly: Add support for custom VCL configuration ([#6662](https://github.com/hashicorp/terraform/issues/6662))
 * provider/google: Support optional uuid naming for Instance Template ([#6604](https://github.com/hashicorp/terraform/issues/6604))
 * provider/openstack: Add support for client certificate authentication ([#6279](https://github.com/hashicorp/terraform/issues/6279))
 * provider/openstack: Allow Neutron-based Floating IP to target a specific tenant ([#6454](https://github.com/hashicorp/terraform/issues/6454))
 * provider/openstack: Enable DHCP By Default ([#6838](https://github.com/hashicorp/terraform/issues/6838))
 * provider/openstack: Implement fixed_ip on Neutron floating ip allocations ([#6837](https://github.com/hashicorp/terraform/issues/6837))
 * provider/openstack: Increase timeouts for image resize, subnets, and routers ([#6764](https://github.com/hashicorp/terraform/issues/6764))
 * provider/openstack: Add `lb_provider` argument to `lb_pool_v1` resource ([#6919](https://github.com/hashicorp/terraform/issues/6919))
 * provider/openstack: Enforce `ForceNew` on Instance Block Device ([#6921](https://github.com/hashicorp/terraform/issues/6921))
 * provider/openstack: Can now stop instances before destroying them ([#7184](https://github.com/hashicorp/terraform/issues/7184))
 * provider/openstack: Disassociate LBaaS v1 Monitors from Pool Before Deletion ([#6997](https://github.com/hashicorp/terraform/issues/6997))
 * provider/powerdns: Add support for PowerDNS 4 API ([#7819](https://github.com/hashicorp/terraform/issues/7819))
 * provider/triton: add `triton_machine` `domain names` ([#7149](https://github.com/hashicorp/terraform/issues/7149))
 * provider/vsphere: Add support for `controller_type` to `vsphere_virtual_machine` ([#6785](https://github.com/hashicorp/terraform/issues/6785))
 * provider/vsphere: Fix bug with `vsphere_virtual_machine` wait for ip ([#6377](https://github.com/hashicorp/terraform/issues/6377))
 * provider/vsphere: Virtual machine update disk ([#6619](https://github.com/hashicorp/terraform/issues/6619))
 * provider/vsphere: `vsphere_virtual_machine` adding controller creation logic ([#6853](https://github.com/hashicorp/terraform/issues/6853))
 * provider/vsphere: `vsphere_virtual_machine` added support for `mac address` on `network_interface` ([#6966](https://github.com/hashicorp/terraform/issues/6966))
 * provider/vsphere: Enhanced `vsphere` logging capabilities ([#6893](https://github.com/hashicorp/terraform/issues/6893))
 * provider/vsphere: Add DiskEnableUUID option to `vsphere_virtual_machine` ([#7088](https://github.com/hashicorp/terraform/issues/7088))
 * provider/vsphere: Virtual Machine and File resources handle Read errors properley ([#7220](https://github.com/hashicorp/terraform/issues/7220))
 * provider/vsphere: set uuid as `vsphere_virtual_machine` output ([#4382](https://github.com/hashicorp/terraform/issues/4382))
 * provider/vsphere: Add support for `keep_on_remove` to `vsphere_virtual_machine` ([#7169](https://github.com/hashicorp/terraform/issues/7169))
 * provider/vsphere: Add support for additional `vsphere_virtial_machine` SCSI controller types ([#7525](https://github.com/hashicorp/terraform/issues/7525))
 * provisioner/file: File provisioners may now have file content set as an attribute ([#7561](https://github.com/hashicorp/terraform/issues/7561))

BUG FIXES:

 * core: Correct the previous fix for a bug causing "attribute not found" messages during destroy, as it was insufficient ([#6599](https://github.com/hashicorp/terraform/issues/6599))
 * core: Fix issue causing syntax errors interpolating count attribute when value passed between modules ([#6833](https://github.com/hashicorp/terraform/issues/6833))
 * core: Fix "diffs didn't match during apply" error for computed sets ([#7205](https://github.com/hashicorp/terraform/issues/7205))
 * core: Fix issue where `terraform init .` would truncate existing files ([#7273](https://github.com/hashicorp/terraform/issues/7273))
 * core: Don't compare diffs between maps with computed values ([#7249](https://github.com/hashicorp/terraform/issues/7249))
 * core: Don't copy existing files over themselves when fetching modules ([#7273](https://github.com/hashicorp/terraform/issues/7273))
 * core: Always increment the state serial number when upgrading the version ([#7402](https://github.com/hashicorp/terraform/issues/7402))
 * core: Fix a crash during eval when we're upgrading an empty state ([#7403](https://github.com/hashicorp/terraform/issues/7403))
 * core: Honor the `-state-out` flag when applying with a plan file ([#7443](https://github.com/hashicorp/terraform/issues/7443))
 * core: Fix a panic when a `terraform_remote_state` data source doesn't exist ([#7464](https://github.com/hashicorp/terraform/issues/7464))
 * core: Fix issue where `ignore_changes` caused incorrect diffs on dependent resources ([#7563](https://github.com/hashicorp/terraform/issues/7563))
 * provider/aws: Manual changes to `aws_codedeploy_deployment_group` resources are now detected ([#7530](https://github.com/hashicorp/terraform/issues/7530))
 * provider/aws: Changing keys in `aws_dynamodb_table` correctly force new resources ([#6829](https://github.com/hashicorp/terraform/issues/6829))
 * provider/aws: Fix a bug where CloudWatch alarms are created repeatedly if the user does not have permission to use the the DescribeAlarms operation ([#7227](https://github.com/hashicorp/terraform/issues/7227))
 * provider/aws: Fix crash in `aws_elasticache_parameter_group` occuring following edits in the console ([#6687](https://github.com/hashicorp/terraform/issues/6687))
 * provider/aws: Fix issue reattaching a VPN gateway to a VPC ([#6987](https://github.com/hashicorp/terraform/issues/6987))
 * provider/aws: Fix issue with Root Block Devices and encrypted flag in Launch Configurations ([#6512](https://github.com/hashicorp/terraform/issues/6512))
 * provider/aws: If more ENIs are attached to `aws_instance`, the one w/ DeviceIndex `0` is always used in context of `aws_instance` (previously unpredictable) ([#6761](https://github.com/hashicorp/terraform/issues/6761))
 * provider/aws: Increased lambda event mapping creation timeout ([#7657](https://github.com/hashicorp/terraform/issues/7657))
 * provider/aws: Handle spurious failures in resourceAwsSecurityGroupRuleRead ([#7377](https://github.com/hashicorp/terraform/issues/7377))
 * provider/aws: Make 'stage_name' required in api_gateway_deployment ([#6797](https://github.com/hashicorp/terraform/issues/6797))
 * provider/aws: Mark Lambda function as gone when it's gone ([#6924](https://github.com/hashicorp/terraform/issues/6924))
 * provider/aws: Trim trailing `.` from `name` in `aws_route53_record` resources to prevent spurious diffs ([#6592](https://github.com/hashicorp/terraform/issues/6592))
 * provider/aws: Update Lambda functions on name change ([#7081](https://github.com/hashicorp/terraform/issues/7081))
 * provider/aws: Updating state when `aws_sns_topic_subscription` is missing ([#6629](https://github.com/hashicorp/terraform/issues/6629))
 * provider/aws: `aws_codedeploy_deployment_group` panic when setting `on_premises_instance_tag_filter` ([#6617](https://github.com/hashicorp/terraform/issues/6617))
 * provider/aws: `aws_db_instance` now defaults `publicly_accessible` to false ([#7117](https://github.com/hashicorp/terraform/issues/7117))
 * provider/aws: `aws_opsworks_application.app_source` SSH key is write-only ([#6649](https://github.com/hashicorp/terraform/issues/6649))
 * provider/aws: fix Elastic Beanstalk `cname_prefix` continual plans ([#6653](https://github.com/hashicorp/terraform/issues/6653))
 * provider/aws: Bundle IOPs and Allocated Storage update for DB Instances ([#7203](https://github.com/hashicorp/terraform/issues/7203))
 * provider/aws: Fix case when instanceId is absent in network interfaces ([#6851](https://github.com/hashicorp/terraform/issues/6851))
 * provider/aws: fix aws_security_group_rule refresh ([#6730](https://github.com/hashicorp/terraform/issues/6730))
 * provider/aws: Fix issue with Elastic Beanstalk and invalid settings ([#7222](https://github.com/hashicorp/terraform/issues/7222))
 * provider/aws: Fix issue where aws_app_cookie_stickiness_policy fails on destroy if LoadBalancer doesn't exist ([#7166](https://github.com/hashicorp/terraform/issues/7166))
 * provider/aws: Stickiness Policy exists, but isn't assigned to the ELB ([#7188](https://github.com/hashicorp/terraform/issues/7188))
 * provider/aws: Fix issue with `manage_bundler` on `aws_opsworks_layers` ([#7219](https://github.com/hashicorp/terraform/issues/7219))
 * provider/aws: Set Elastic Beanstalk stack name back to state ([#7445](https://github.com/hashicorp/terraform/issues/7445))
 * provider/aws: Allow recreation of VPC Peering Connection when state is rejected ([#7466](https://github.com/hashicorp/terraform/issues/7466))
 * provider/aws: Remove EFS File System from State when NotFound ([#7437](https://github.com/hashicorp/terraform/issues/7437))
 * provider/aws: `aws_customer_gateway` refreshing from state on deleted state ([#7482](https://github.com/hashicorp/terraform/issues/7482))
 * provider/aws: Retry finding `aws_route` after creating it ([#7463](https://github.com/hashicorp/terraform/issues/7463))
 * provider/aws: Refresh CloudWatch Group from state on 404 ([#7576](https://github.com/hashicorp/terraform/issues/7576))
 * provider/aws: Adding in additional retry logic due to latency with delete of `db_option_group` ([#7312](https://github.com/hashicorp/terraform/issues/7312))
 * provider/aws: Safely get ELB values ([#7585](https://github.com/hashicorp/terraform/issues/7585))
 * provider/aws: Fix bug for recurring plans on ec2-classic and vpc in beanstalk ([#6491](https://github.com/hashicorp/terraform/issues/6491))
 * provider/aws: Bump rds_cluster timeout to 15 mins ([#7604](https://github.com/hashicorp/terraform/issues/7604))
 * provider/aws: Fix ICMP fields in `aws_network_acl_rule` to allow ICMP code 0 (echo reply) to be configured ([#7669](https://github.com/hashicorp/terraform/issues/7669))
 * provider/aws: Fix bug with Updating `aws_autoscaling_group` `enabled_metrics` ([#7698](https://github.com/hashicorp/terraform/issues/7698))
 * provider/aws: Ignore IOPS on non io1 AWS root_block_device ([#7783](https://github.com/hashicorp/terraform/issues/7783))
 * provider/aws: Ignore missing ENI attachment when trying to detach ENI ([#7185](https://github.com/hashicorp/terraform/issues/7185))
 * provider/aws: Fix issue updating ElasticBeanstalk Environment templates ([#7811](https://github.com/hashicorp/terraform/issues/7811))
 * provider/aws: Restore Defaults to SQS Queues ([#7818](https://github.com/hashicorp/terraform/issues/7818))
 * provider/aws: Don't delete Lambda function from state on initial call of the Read func ([#7829](https://github.com/hashicorp/terraform/issues/7829))
 * provider/aws: `aws_vpn_gateway` should be removed from state when in deleted state ([#7861](https://github.com/hashicorp/terraform/issues/7861))
 * provider/aws: Fix aws_route53_record 0-2 migration ([#7907](https://github.com/hashicorp/terraform/issues/7907))
 * provider/azurerm: Fixes terraform crash when using SSH keys with `azurerm_virtual_machine` ([#6766](https://github.com/hashicorp/terraform/issues/6766))
 * provider/azurerm: Fix a bug causing 'diffs do not match' on `azurerm_network_interface` resources ([#6790](https://github.com/hashicorp/terraform/issues/6790))
 * provider/azurerm: Normalizes `availability_set_id` casing to avoid spurious diffs in `azurerm_virtual_machine` ([#6768](https://github.com/hashicorp/terraform/issues/6768))
 * provider/azurerm: Add support for storage container name validation ([#6852](https://github.com/hashicorp/terraform/issues/6852))
 * provider/azurerm: Remove storage containers and blobs when storage accounts are not found ([#6855](https://github.com/hashicorp/terraform/issues/6855))
 * provider/azurerm: `azurerm_virtual_machine` fix `additional_unattend_rm` Windows config option ([#7105](https://github.com/hashicorp/terraform/issues/7105))
 * provider/azurerm: Fix `azurerm_virtual_machine` windows_config ([#7123](https://github.com/hashicorp/terraform/issues/7123))
 * provider/azurerm: `azurerm_dns_cname_record` can create CNAME records again ([#7113](https://github.com/hashicorp/terraform/issues/7113))
 * provider/azurerm: `azurerm_network_security_group` now waits for the provisioning state of `ready` before proceeding ([#7307](https://github.com/hashicorp/terraform/issues/7307))
 * provider/azurerm: `computer_name` is now required for `azurerm_virtual_machine` resources ([#7308](https://github.com/hashicorp/terraform/issues/7308))
 * provider/azurerm: destroy azurerm_virtual_machine OS Disk VHD on deletion ([#7584](https://github.com/hashicorp/terraform/issues/7584))
 * provider/azurerm: catch `azurerm_template_deployment` erroring silently ([#7644](https://github.com/hashicorp/terraform/issues/7644))
 * provider/azurerm: changing the name of an `azurerm_virtual_machine` now forces a new resource ([#7646](https://github.com/hashicorp/terraform/issues/7646))
 * provider/azurerm: azurerm_storage_account now returns storage keys value instead of their names ([#7674](https://github.com/hashicorp/terraform/issues/7674))
 * provider/azurerm: `azurerm_virtual_machine` computer_name now Required ([#7308](https://github.com/hashicorp/terraform/issues/7308))
 * provider/azurerm: Change of `availability_set_id` on `azurerm_virtual_machine` should ForceNew ([#7650](https://github.com/hashicorp/terraform/issues/7650))
 * provider/azurerm: Wait for `azurerm_storage_account` to be available ([#7329](https://github.com/hashicorp/terraform/issues/7329))
 * provider/cloudflare: Fix issue upgrading CloudFlare Records created before v0.6.15 ([#6969](https://github.com/hashicorp/terraform/issues/6969))
 * provider/cloudstack: Fix using `cloudstack_network_acl` within a project ([#6743](https://github.com/hashicorp/terraform/issues/6743))
 * provider/cloudstack: Fix refresing `cloudstack_network_acl_rule` when the associated ACL is deleted ([#7612](https://github.com/hashicorp/terraform/issues/7612))
 * provider/cloudstack: Fix refresing `cloudstack_port_forward` when the associated IP address is no longer associated ([#7612](https://github.com/hashicorp/terraform/issues/7612))
 * provider/cloudstack: Fix creating `cloudstack_network` with offerings that do not support specifying IP ranges ([#7612](https://github.com/hashicorp/terraform/issues/7612))
 * provider/digitalocean: Stop `digitocean_droplet` forcing new resource on uppercase region ([#7044](https://github.com/hashicorp/terraform/issues/7044))
 * provider/digitalocean: Reassign Floating IP when droplet changes ([#7411](https://github.com/hashicorp/terraform/issues/7411))
 * provider/google: Fix a bug causing an error attempting to delete an already-deleted `google_compute_disk` ([#6689](https://github.com/hashicorp/terraform/issues/6689))
 * provider/mysql: Specifying empty provider credentials no longer causes a panic ([#7211](https://github.com/hashicorp/terraform/issues/7211))
 * provider/openstack: Reassociate Floating IP on network changes ([#6579](https://github.com/hashicorp/terraform/issues/6579))
 * provider/openstack: Ensure CIDRs Are Lower Case ([#6864](https://github.com/hashicorp/terraform/issues/6864))
 * provider/openstack: Rebuild Instances On Network Changes ([#6844](https://github.com/hashicorp/terraform/issues/6844))
 * provider/openstack: Firewall rules are applied in the correct order ([#7194](https://github.com/hashicorp/terraform/issues/7194))
 * provider/openstack: Fix Security Group EOF Error when Adding / Removing Multiple Groups ([#7468](https://github.com/hashicorp/terraform/issues/7468))
 * provider/openstack: Fixing boot volumes interfering with block storage volumes list ([#7649](https://github.com/hashicorp/terraform/issues/7649))
 * provider/vsphere: `gateway` and `ipv6_gateway` are now read from `vsphere_virtual_machine` resources ([#6522](https://github.com/hashicorp/terraform/issues/6522))
 * provider/vsphere: `ipv*_gateway` parameters won't force a new `vsphere_virtual_machine` ([#6635](https://github.com/hashicorp/terraform/issues/6635))
 * provider/vsphere: adding a `vsphere_virtual_machine` migration ([#7023](https://github.com/hashicorp/terraform/issues/7023))
 * provider/vsphere: Don't require vsphere debug paths to be set ([#7027](https://github.com/hashicorp/terraform/issues/7027))
 * provider/vsphere: Fix bug where `enable_disk_uuid` was not set on `vsphere_virtual_machine` resources ([#7275](https://github.com/hashicorp/terraform/issues/7275))
 * provider/vsphere: Make `vsphere_virtual_machine` `product_key` optional ([#7410](https://github.com/hashicorp/terraform/issues/7410))
 * provider/vsphere: Refreshing devices list after adding a disk or cdrom controller ([#7167](https://github.com/hashicorp/terraform/issues/7167))
 * provider/vsphere: `vsphere_virtual_machine` no longer has to be powered on to delete ([#7206](https://github.com/hashicorp/terraform/issues/7206))
 * provider/vSphere: Fixes the hasBootableVmdk flag when attaching multiple disks ([#7804](https://github.com/hashicorp/terraform/issues/7804))
 * provisioner/remote-exec: Properly seed random script paths so they are not deterministic across runs ([#7413](https://github.com/hashicorp/terraform/issues/7413))

## 0.6.16 (May 9, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

 * provider/aws: `aws_eip` field `private_ip` is now a computed value, and cannot be set in your configuration.
    Use `associate_with_private_ip` instead. See ([#6521](https://github.com/hashicorp/terraform/issues/6521))

FEATURES:

 * **New provider:** `librato` ([#3371](https://github.com/hashicorp/terraform/issues/3371))
 * **New provider:** `softlayer` ([#4327](https://github.com/hashicorp/terraform/issues/4327))
 * **New resource:** `aws_api_gateway_account` ([#6321](https://github.com/hashicorp/terraform/issues/6321))
 * **New resource:** `aws_api_gateway_authorizer` ([#6320](https://github.com/hashicorp/terraform/issues/6320))
 * **New resource:** `aws_db_event_subscription` ([#6367](https://github.com/hashicorp/terraform/issues/6367))
 * **New resource:** `aws_db_option_group` ([#4401](https://github.com/hashicorp/terraform/issues/4401))
 * **New resource:** `aws_eip_association` ([#6552](https://github.com/hashicorp/terraform/issues/6552))
 * **New resource:** `openstack_networking_secgroup_rule_v2` ([#6410](https://github.com/hashicorp/terraform/issues/6410))
 * **New resource:** `openstack_networking_secgroup_v2` ([#6410](https://github.com/hashicorp/terraform/issues/6410))
 * **New resource:** `vsphere_file` ([#6401](https://github.com/hashicorp/terraform/issues/6401))

IMPROVEMENTS:

 * core: update HCL dependency to improve whitespace handling in `terraform fmt` ([#6347](https://github.com/hashicorp/terraform/issues/6347))
 * core: Add support for marking outputs as sensitive ([#6559](https://github.com/hashicorp/terraform/issues/6559))
 * provider/aws: Add agent_version argument to `aws_opswork_stack` ([#6493](https://github.com/hashicorp/terraform/issues/6493))
 * provider/aws: Add support for request parameters to `api_gateway_method` & `api_gateway_integration` ([#6501](https://github.com/hashicorp/terraform/issues/6501))
 * provider/aws: Add support for response parameters to `api_gateway_method_response` & `api_gateway_integration_response` ([#6344](https://github.com/hashicorp/terraform/issues/6344))
 * provider/aws: Allow empty S3 config in Cloudfront Origin ([#6487](https://github.com/hashicorp/terraform/issues/6487))
 * provider/aws: Improve error handling in IAM Server Certificates ([#6442](https://github.com/hashicorp/terraform/issues/6442))
 * provider/aws: Use `sts:GetCallerIdentity` as additional method for getting AWS account ID ([#6385](https://github.com/hashicorp/terraform/issues/6385))
 * provider/aws: `aws_redshift_cluster` `automated_snapshot_retention_period` didn't allow 0 value ([#6537](https://github.com/hashicorp/terraform/issues/6537))
 * provider/aws: Add CloudFront `hosted_zone_id` attribute ([#6530](https://github.com/hashicorp/terraform/issues/6530))
 * provider/azurerm: Increase timeout for ARM Template deployments to 40 minutes ([#6319](https://github.com/hashicorp/terraform/issues/6319))
 * provider/azurerm: Make `private_ip_address` an exported field on `azurerm_network_interface` ([#6538](https://github.com/hashicorp/terraform/issues/6538))
 * provider/azurerm: Add support for `tags` to `azurerm_virtual_machine` ([#6556](https://github.com/hashicorp/terraform/issues/6556))
 * provider/azurerm: Add `os_type` and `image_uri` in `azurerm_virtual_machine` ([#6553](https://github.com/hashicorp/terraform/issues/6553))
 * provider/cloudflare: Add proxied option to `cloudflare_record` ([#5508](https://github.com/hashicorp/terraform/issues/5508))
 * provider/docker: Add ability to keep docker image locally on terraform destroy ([#6376](https://github.com/hashicorp/terraform/issues/6376))
 * provider/fastly: Add S3 Log Streaming to Fastly Service ([#6378](https://github.com/hashicorp/terraform/issues/6378))
 * provider/fastly: Add Conditions to Fastly Service ([#6481](https://github.com/hashicorp/terraform/issues/6481))
 * provider/github: Add support for Github Enterprise via base_url configuration option ([#6434](https://github.com/hashicorp/terraform/issues/6434))
 * provider/triton: Add support for specifying network interfaces on `triton machine` resources ([#6418](https://github.com/hashicorp/terraform/issues/6418))
 * provider/triton: Deleted firewall rules no longer prevent refresh ([#6529](https://github.com/hashicorp/terraform/issues/6529))
 * provider/vsphere: Add `skip_customization` option to `vsphere_virtual_machine` resources ([#6355](https://github.com/hashicorp/terraform/issues/6355))
 * provider/vsphere: Add ability to specify and mount bootable vmdk in `vsphere_virtual_machine` ([#6146](https://github.com/hashicorp/terraform/issues/6146))
 * provider/vsphere: Add support for IPV6 to `vsphere_virtual_machine` ([#6457](https://github.com/hashicorp/terraform/issues/6457))
 * provider/vsphere: Add support for `memory_reservation` to `vsphere_virtual_machine` ([#6036](https://github.com/hashicorp/terraform/issues/6036))
 * provider/vsphere: Checking for empty diskPath in `vsphere_virtual_machine` before creating ([#6400](https://github.com/hashicorp/terraform/issues/6400))
 * provider/vsphere: Support updates to vcpu and memory on `vsphere_virtual_machine` ([#6356](https://github.com/hashicorp/terraform/issues/6356))
 * remote/s3: Logic for loading credentials now follows the same [conventions as AWS provider](https://www.terraform.io/docs/providers/aws/index.html#authentication) which means it also supports EC2 role auth and session token (e.g. assumed IAM Roles) ([#5270](https://github.com/hashicorp/terraform/issues/5270))

BUG FIXES:

 * core: Boolean values in diffs are normalized to `true` and `false`, eliminating some erroneous diffs ([#6499](https://github.com/hashicorp/terraform/issues/6499))
 * core: Fix a bug causing "attribute not found" messages during destroy ([#6557](https://github.com/hashicorp/terraform/issues/6557))
 * provider/aws: Allow account ID checks on EC2 instances & w/ federated accounts ([#5030](https://github.com/hashicorp/terraform/issues/5030))
 * provider/aws: Fix an eventually consistent issue aws_security_group_rule and possible duplications ([#6325](https://github.com/hashicorp/terraform/issues/6325))
 * provider/aws: Fix bug where `aws_elastic_beanstalk_environment` ignored `wait_for_ready_timeout` ([#6358](https://github.com/hashicorp/terraform/issues/6358))
 * provider/aws: Fix bug where `aws_elastic_beanstalk_environment` update config template didn't work ([#6342](https://github.com/hashicorp/terraform/issues/6342))
 * provider/aws: Fix issue in updating CloudFront distribution LoggingConfig ([#6407](https://github.com/hashicorp/terraform/issues/6407))
 * provider/aws: Fix issue in upgrading AutoScaling Policy to use `min_adjustment_magnitude` ([#6440](https://github.com/hashicorp/terraform/issues/6440))
 * provider/aws: Fix issue replacing Network ACL Relationship ([#6421](https://github.com/hashicorp/terraform/issues/6421))
 * provider/aws: Fix issue with KMS Alias keys and name prefixes ([#6328](https://github.com/hashicorp/terraform/issues/6328))
 * provider/aws: Fix issue with encrypted snapshots of block devices in `aws_launch_configuration` resources ([#6452](https://github.com/hashicorp/terraform/issues/6452))
 * provider/aws: Fix read of `aws_cloudwatch_log_group` after an update is applied ([#6384](https://github.com/hashicorp/terraform/issues/6384))
 * provider/aws: Fix updating `number_of_nodes` on `aws_redshift_cluster` ([#6333](https://github.com/hashicorp/terraform/issues/6333))
 * provider/aws: Omit `aws_cloudfront_distribution` custom_error fields when not explicitly set ([#6382](https://github.com/hashicorp/terraform/issues/6382))
 * provider/aws: Refresh state on `aws_sqs_queue` not found ([#6381](https://github.com/hashicorp/terraform/issues/6381))
 * provider/aws: Respect `selection_pattern` in `aws_api_gateway_integration_response` (previously ignored field) ([#5893](https://github.com/hashicorp/terraform/issues/5893))
 * provider/aws: `aws_cloudfront_distribution` resources now require the `cookies` argument ([#6505](https://github.com/hashicorp/terraform/issues/6505))
 * provider/aws: `aws_route` crash when used with `aws_vpc_endpoint` ([#6338](https://github.com/hashicorp/terraform/issues/6338))
 * provider/aws: validate `cluster_id` length for `aws_elasticache_cluster` ([#6330](https://github.com/hashicorp/terraform/issues/6330))
 * provider/azurerm: `ssh_keys` can now be set for `azurerm_virtual_machine` resources, allowing provisioning ([#6541](https://github.com/hashicorp/terraform/issues/6541))
 * provider/azurerm: Fix issue that updating `azurerm_virtual_machine` was failing due to empty adminPassword ([#6528](https://github.com/hashicorp/terraform/issues/6528))
 * provider/azurerm: `storage_data_disk` settings now work correctly on `azurerm_virtual_machine` resources ([#6543](https://github.com/hashicorp/terraform/issues/6543))
 * provider/cloudflare: can manage apex records ([#6449](https://github.com/hashicorp/terraform/issues/6449))
 * provider/cloudflare: won't refresh with incorrect record if names match ([#6449](https://github.com/hashicorp/terraform/issues/6449))
 * provider/datadog: `notify_no_data` and `no_data_timeframe` are set correctly for `datadog_monitor` resources ([#6509](https://github.com/hashicorp/terraform/issues/6509))
 * provider/docker: Fix crash when using empty string in the `command` list in `docker_container` resources ([#6424](https://github.com/hashicorp/terraform/issues/6424))
 * provider/vsphere: Memory reservations are now set correctly in `vsphere_virtual_machine` resources ([#6482](https://github.com/hashicorp/terraform/issues/6482))

## 0.6.15 (April 22, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:
 * `aws_instance` - if you still use `security_groups` field for SG IDs - i.e. inside VPC, this will generate diffs during `plan` and `apply` will **recreate** the resource. Terraform expects IDs (VPC SGs) inside `vpc_security_group_ids`.

FEATURES:

 * **New command:** `terraform fmt` to automatically normalize config file style ([#4955](https://github.com/hashicorp/terraform/issues/4955))
 * **New interpolation function:** `jsonencode` ([#5890](https://github.com/hashicorp/terraform/issues/5890))
 * **New provider:** `cobbler` ([#5969](https://github.com/hashicorp/terraform/issues/5969))
 * **New provider:** `fastly` ([#5814](https://github.com/hashicorp/terraform/issues/5814))
 * **New resource:** `aws_cloudfront_distribution` ([#5221](https://github.com/hashicorp/terraform/issues/5221))
 * **New resource:** `aws_cloudfront_origin_access_identity` ([#5221](https://github.com/hashicorp/terraform/issues/5221))
 * **New resource:** `aws_iam_user_ssh_key` ([#5774](https://github.com/hashicorp/terraform/issues/5774))
 * **New resource:** `aws_s3_bucket_notification` ([#5473](https://github.com/hashicorp/terraform/issues/5473))
 * **New resource:** `cloudstack_static_nat` ([#6004](https://github.com/hashicorp/terraform/issues/6004))
 * **New resource:** `consul_key_prefix` ([#5988](https://github.com/hashicorp/terraform/issues/5988))
 * **New resource:** `aws_default_network_acl` ([#6165](https://github.com/hashicorp/terraform/issues/6165))
 * **New resource:** `triton_fabric` ([#5920](https://github.com/hashicorp/terraform/issues/5920))
 * **New resource:** `triton_vlan` ([#5920](https://github.com/hashicorp/terraform/issues/5920))
 * **New resource:** `aws_opsworks_application` ([#4419](https://github.com/hashicorp/terraform/issues/4419))
 * **New resource:** `aws_opsworks_instance` ([#4276](https://github.com/hashicorp/terraform/issues/4276))
 * **New resource:** `aws_cloudwatch_log_subscription_filter` ([#5996](https://github.com/hashicorp/terraform/issues/5996))
 * **New resource:** `openstack_networking_router_route_v2` ([#6207](https://github.com/hashicorp/terraform/issues/6207))

IMPROVEMENTS:

 * command/apply: Output will now show periodic status updates of slow resources. ([#6163](https://github.com/hashicorp/terraform/issues/6163))
 * core: Variables passed between modules are now type checked ([#6185](https://github.com/hashicorp/terraform/issues/6185))
 * core: Smaller release binaries by stripping debug information ([#6238](https://github.com/hashicorp/terraform/issues/6238))
 * provider/aws: Add support for Step Scaling in `aws_autoscaling_policy` ([#4277](https://github.com/hashicorp/terraform/issues/4277))
 * provider/aws: Add support for `cname_prefix` to `aws_elastic_beanstalk_environment` resource ([#5966](https://github.com/hashicorp/terraform/issues/5966))
 * provider/aws: Add support for trigger_configuration to `aws_codedeploy_deployment_group` ([#5599](https://github.com/hashicorp/terraform/issues/5599))
 * provider/aws: Adding outputs for elastic_beanstalk_environment resource ([#5915](https://github.com/hashicorp/terraform/issues/5915))
 * provider/aws: Adds `wait_for_ready_timeout` option to `aws_elastic_beanstalk_environment` ([#5967](https://github.com/hashicorp/terraform/issues/5967))
 * provider/aws: Allow `aws_db_subnet_group` description to be updated ([#5921](https://github.com/hashicorp/terraform/issues/5921))
 * provider/aws: Allow multiple EIPs to associate to single ENI ([#6070](https://github.com/hashicorp/terraform/issues/6070))
 * provider/aws: Change `aws_elb` access_logs to list type ([#5065](https://github.com/hashicorp/terraform/issues/5065))
 * provider/aws: Check that InternetGateway exists before returning from creation ([#6105](https://github.com/hashicorp/terraform/issues/6105))
 * provider/aws: Don't Base64-encode EC2 userdata if it is already Base64 encoded ([#6140](https://github.com/hashicorp/terraform/issues/6140))
 * provider/aws: Making the Cloudwatch Event Rule Target `target_id` optional ([#5787](https://github.com/hashicorp/terraform/issues/5787))
 * provider/aws: Timeouts for `elasticsearch_domain` are increased ([#5910](https://github.com/hashicorp/terraform/issues/5910))
 * provider/aws: `aws_codecommit_repository` set `default_branch` only if defined ([#5904](https://github.com/hashicorp/terraform/issues/5904))
 * provider/aws: `aws_redshift_cluster` allows usernames with underscore in it ([#5935](https://github.com/hashicorp/terraform/issues/5935))
 * provider/aws: normalise json for `aws_sns_topic` ([#6089](https://github.com/hashicorp/terraform/issues/6089))
 * provider/aws: normalize json for `aws_cloudwatch_event_rule` ([#6025](https://github.com/hashicorp/terraform/issues/6025))
 * provider/aws: increase timeout for aws_redshift_cluster ([#6305](https://github.com/hashicorp/terraform/issues/6305))
 * provider/aws: Opsworks layers now support `custom_json` argument ([#4272](https://github.com/hashicorp/terraform/issues/4272))
 * provider/aws: Added migration for `tier` attribute in `aws_elastic_beanstalk_environment` ([#6167](https://github.com/hashicorp/terraform/issues/6167))
 * provider/aws: Use resource.Retry for route creation and deletion ([#6225](https://github.com/hashicorp/terraform/issues/6225))
 * provider/aws: Add support S3 Bucket Lifecycle Rule ([#6220](https://github.com/hashicorp/terraform/issues/6220))
 * provider/clc: Override default `account` alias in provider config ([#5785](https://github.com/hashicorp/terraform/issues/5785))
 * provider/cloudstack: Deprecate `ipaddress` in favour of `ip_address` in all resources ([#6010](https://github.com/hashicorp/terraform/issues/6010))
 * provider/cloudstack: Deprecate allowing names (instead of IDs) for parameters that reference other resources ([#6123](https://github.com/hashicorp/terraform/issues/6123))
 * provider/datadog: Add heredoc support to message, escalation_message, and query ([#5788](https://github.com/hashicorp/terraform/issues/5788))
 * provider/docker: Add support for docker run --user option ([#5300](https://github.com/hashicorp/terraform/issues/5300))
 * provider/github: Add support for privacy to `github_team` ([#6116](https://github.com/hashicorp/terraform/issues/6116))
 * provider/google: Accept GOOGLE_CLOUD_KEYFILE_JSON env var for credentials ([#6007](https://github.com/hashicorp/terraform/issues/6007))
 * provider/google: Add "project" argument and attribute to all GCP compute resources which inherit from the provider's value ([#6112](https://github.com/hashicorp/terraform/issues/6112))
 * provider/google: Make "project" attribute on provider configuration optional ([#6112](https://github.com/hashicorp/terraform/issues/6112))
 * provider/google: Read more common configuration values from the environment and clarify precedence ordering ([#6114](https://github.com/hashicorp/terraform/issues/6114))
 * provider/google: `addons_config` and `subnetwork` added as attributes to `google_container_cluster` ([#5871](https://github.com/hashicorp/terraform/issues/5871))
 * provider/fastly: Add support for Request Headers ([#6197](https://github.com/hashicorp/terraform/issues/6197))
 * provider/fastly: Add support for Gzip rules ([#6247](https://github.com/hashicorp/terraform/issues/6247))
 * provider/openstack: Add value_specs argument and attribute for routers ([#4898](https://github.com/hashicorp/terraform/issues/4898))
 * provider/openstack: Allow subnets with no gateway ([#6060](https://github.com/hashicorp/terraform/issues/6060))
 * provider/openstack: Enable Token Authentication ([#6081](https://github.com/hashicorp/terraform/issues/6081))
 * provider/postgresql: New `ssl_mode` argument allowing different SSL usage tradeoffs ([#6008](https://github.com/hashicorp/terraform/issues/6008))
 * provider/vsphere: Support for linked clones and Windows-specific guest config options ([#6087](https://github.com/hashicorp/terraform/issues/6087))
 * provider/vsphere: Checking for Powered Off State before `vsphere_virtual_machine` deletion ([#6283](https://github.com/hashicorp/terraform/issues/6283))
 * provider/vsphere: Support mounting ISO images to virtual cdrom drives ([#4243](https://github.com/hashicorp/terraform/issues/4243))
 * provider/vsphere: Fix missing ssh connection info ([#4283](https://github.com/hashicorp/terraform/issues/4283))
 * provider/google: Deprecate unused "region" attribute in `global_forwarding_rule`; this attribute was never used anywhere in the computation of the resource ([#6112](https://github.com/hashicorp/terraform/issues/6112))
 * provider/cloudstack: Add group attribute to `cloudstack_instance` resource ([#6023](https://github.com/hashicorp/terraform/issues/6023))
 * provider/azurerm: Provider meaningful error message when credentials not correct ([#6290](https://github.com/hashicorp/terraform/issues/6290))
 * provider/cloudstack: Improve support for using projects ([#6282](https://github.com/hashicorp/terraform/issues/6282))

BUG FIXES:

 * core: Providers are now correctly inherited down a nested module tree ([#6186](https://github.com/hashicorp/terraform/issues/6186))
 * provider/aws: Convert protocols to standard format for Security Groups ([#5881](https://github.com/hashicorp/terraform/issues/5881))
 * provider/aws: Fix Lambda VPC integration (missing `vpc_id` field in schema) ([#6157](https://github.com/hashicorp/terraform/issues/6157))
 * provider/aws: Fix `aws_route panic` when destination CIDR block is nil ([#5781](https://github.com/hashicorp/terraform/issues/5781))
 * provider/aws: Fix issue re-creating deleted VPC peering connections ([#5959](https://github.com/hashicorp/terraform/issues/5959))
 * provider/aws: Fix issue with changing iops when also changing storage type to io1 on RDS ([#5676](https://github.com/hashicorp/terraform/issues/5676))
 * provider/aws: Fix issue with retrying deletion of Network ACLs ([#5954](https://github.com/hashicorp/terraform/issues/5954))
 * provider/aws: Fix potential crash when receiving malformed `aws_route` API responses ([#5867](https://github.com/hashicorp/terraform/issues/5867))
 * provider/aws: Guard against empty responses from Lambda Permissions ([#5838](https://github.com/hashicorp/terraform/issues/5838))
 * provider/aws: Normalize and compact SQS Redrive, Policy JSON ([#5888](https://github.com/hashicorp/terraform/issues/5888))
 * provider/aws: Fix issue updating ElasticBeanstalk Configuraiton Templates ([#6307](https://github.com/hashicorp/terraform/issues/6307))
 * provider/aws: Remove CloudTrail Trail from state if not found ([#6024](https://github.com/hashicorp/terraform/issues/6024))
 * provider/aws: Fix crash in AWS S3 Bucket when website index/error is empty ([#6269](https://github.com/hashicorp/terraform/issues/6269))
 * provider/aws: Report better error message in `aws_route53_record` when `set_identifier` is required ([#5777](https://github.com/hashicorp/terraform/issues/5777))
 * provider/aws: Show human-readable error message when failing to read an EBS volume ([#6038](https://github.com/hashicorp/terraform/issues/6038))
 * provider/aws: set ASG `health_check_grace_period` default to 300 ([#5830](https://github.com/hashicorp/terraform/issues/5830))
 * provider/aws: Fix issue with with Opsworks and empty Custom Cook Book sources ([#6078](https://github.com/hashicorp/terraform/issues/6078))
 * provider/aws: wait for IAM instance profile to propagate when creating Opsworks stacks ([#6049](https://github.com/hashicorp/terraform/issues/6049))
 * provider/aws: Don't read back `aws_opsworks_stack` cookbooks source password ([#6203](https://github.com/hashicorp/terraform/issues/6203))
 * provider/aws: Resolves DefaultOS and ConfigurationManager conflict on `aws_opsworks_stack` ([#6244](https://github.com/hashicorp/terraform/issues/6244))
 * provider/aws: Renaming `aws_elastic_beanstalk_configuration_template``option_settings` to `setting` ([#6043](https://github.com/hashicorp/terraform/issues/6043))
 * provider/aws: `aws_customer_gateway` will properly populate `bgp_asn` on refresh. [no issue]
 * provider/aws: provider/aws: Refresh state on `aws_directory_service_directory` not found ([#6294](https://github.com/hashicorp/terraform/issues/6294))
 * provider/aws: `aws_elb` `cross_zone_load_balancing` is not refreshed in the state file ([#6295](https://github.com/hashicorp/terraform/issues/6295))
 * provider/aws: `aws_autoscaling_group` will properly populate `tag` on refresh. [no issue]
 * provider/azurerm: Fix detection of `azurerm_storage_account` resources removed manually ([#5878](https://github.com/hashicorp/terraform/issues/5878))
 * provider/docker: Docker Image will be deleted on destroy ([#5801](https://github.com/hashicorp/terraform/issues/5801))
 * provider/openstack: Fix Disabling DHCP on Subnets ([#6052](https://github.com/hashicorp/terraform/issues/6052))
 * provider/openstack: Fix resizing when Flavor Name changes ([#6020](https://github.com/hashicorp/terraform/issues/6020))
 * provider/openstack: Fix Access Address Detection ([#6181](https://github.com/hashicorp/terraform/issues/6181))
 * provider/openstack: Fix admin_state_up on openstack_lb_member_v1 ([#6267](https://github.com/hashicorp/terraform/issues/6267))
 * provider/triton: Firewall status on `triton_machine` resources is reflected correctly ([#6119](https://github.com/hashicorp/terraform/issues/6119))
 * provider/triton: Fix time out when applying updates to Triton machine metadata ([#6149](https://github.com/hashicorp/terraform/issues/6149))
 * provider/vsphere: Add error handling to `vsphere_folder` ([#6095](https://github.com/hashicorp/terraform/issues/6095))
 * provider/cloudstack: Fix mashalling errors when using CloudStack 4.7.x (or newer) [GH-#226]

## 0.6.14 (March 21, 2016)

FEATURES:

  * **New provider:** `triton` - Manage Joyent Triton public cloud or on-premise installations ([#5738](https://github.com/hashicorp/terraform/issues/5738))
  * **New provider:** `clc` - Manage CenturyLink Cloud resources ([#4893](https://github.com/hashicorp/terraform/issues/4893))
  * **New provider:** `github` - Manage GitHub Organization permissions with Terraform config ([#5194](https://github.com/hashicorp/terraform/issues/5194))
  * **New provider:** `influxdb` - Manage InfluxDB databases ([#3478](https://github.com/hashicorp/terraform/issues/3478))
  * **New provider:** `ultradns` - Manage UltraDNS records ([#5716](https://github.com/hashicorp/terraform/issues/5716))
  * **New resource:** `aws_cloudwatch_log_metric_filter` ([#5444](https://github.com/hashicorp/terraform/issues/5444))
  * **New resource:** `azurerm_virtual_machine` ([#5514](https://github.com/hashicorp/terraform/issues/5514))
  * **New resource:** `azurerm_template_deployment` ([#5758](https://github.com/hashicorp/terraform/issues/5758))
  * **New interpolation function:** `uuid` ([#5575](https://github.com/hashicorp/terraform/issues/5575))

IMPROVEMENTS:

  * core: provisioners connecting via WinRM now respect HTTPS settings  ([#5761](https://github.com/hashicorp/terraform/issues/5761))
  * provider/aws: `aws_db_instance` now makes `identifier` optional and generates a unique ID when it is omitted ([#5723](https://github.com/hashicorp/terraform/issues/5723))
  * provider/aws: `aws_redshift_cluster` now allows`publicly_accessible` to be modified ([#5721](https://github.com/hashicorp/terraform/issues/5721))
  * provider/aws: `aws_kms_alias` now allows name to be auto-generated with a `name_prefix` ([#5594](https://github.com/hashicorp/terraform/issues/5594))

BUG FIXES:

  * core: Color output is now shown correctly when running Terraform on Windows ([#5718](https://github.com/hashicorp/terraform/issues/5718))
  * core: HEREDOCs can now be indented in line with configuration using `<<-` and hanging indent is removed ([#5740](https://github.com/hashicorp/terraform/issues/5740))
  * core: Invalid HCL syntax of nested object blocks no longer causes a crash ([#5740](https://github.com/hashicorp/terraform/issues/5740))
  * core: Local directory-based modules now use junctions instead of symbolic links on Windows ([#5739](https://github.com/hashicorp/terraform/issues/5739))
  * core: Modules sourced from a Mercurial repository now work correctly on Windows ([#5739](https://github.com/hashicorp/terraform/issues/5739))
  * core: Address some issues with ignore_changes ([#5635](https://github.com/hashicorp/terraform/issues/5635))
  * core: Add a lock to fix an interpolation issue caught by the Go 1.6 concurrent map access detector ([#5772](https://github.com/hashicorp/terraform/issues/5772))
  * provider/aws: Fix crash when an `aws_rds_cluster_instance` is removed outside of Terraform ([#5717](https://github.com/hashicorp/terraform/issues/5717))
  * provider/aws: `aws_cloudformation_stack` use `timeout_in_minutes` for retry timeout to prevent unecessary timeouts ([#5712](https://github.com/hashicorp/terraform/issues/5712))
  * provider/aws: `aws_lambda_function` resources no longer error on refresh if deleted externally to Terraform ([#5668](https://github.com/hashicorp/terraform/issues/5668))
  * provider/aws: `aws_vpn_connection` resources deleted via the console on longer cause a crash ([#5747](https://github.com/hashicorp/terraform/issues/5747))
  * provider/aws: Fix crasher in Elastic Beanstalk Configuration when using options ([#5756](https://github.com/hashicorp/terraform/issues/5756))
  * provider/aws: Fix issue preventing `aws_opsworks_stck` from working with Windows set as the OS ([#5724](https://github.com/hashicorp/terraform/issues/5724))
  * provider/digitalocean: `digitalocean_ssh_key` resources no longer cause a panic if there is no network connectivity ([#5748](https://github.com/hashicorp/terraform/issues/5748))
  * provider/google: Default description `google_dns_managed_zone` resources to "Managed By Terraform" ([#5428](https://github.com/hashicorp/terraform/issues/5428))
  * provider/google: Fix error message on invalid instance URL for `google_compute_instance_group` ([#5715](https://github.com/hashicorp/terraform/issues/5715))
  * provider/vsphere: provide `host` to provisioner connections ([#5558](https://github.com/hashicorp/terraform/issues/5558))
  * provisioner/remote-exec: Address race condition introduced with script cleanup step introduced in 0.6.13 ([#5751](https://github.com/hashicorp/terraform/issues/5751))

## 0.6.13 (March 16, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

  * provider/aws: `aws_s3_bucket_object` field `etag` is now trimming off quotes (returns raw MD5 hash) ([#5305](https://github.com/hashicorp/terraform/issues/5305))
  * provider/aws: `aws_autoscaling_group` now supports metrics collection, so a diff installing the default value of `1Minute` for the `metrics_granularity` field is expected. This diff should resolve in the next `terraform apply` w/ no AWS API calls ([#4688](https://github.com/hashicorp/terraform/issues/4688))
  * provider/consul: `consul_keys` `key` blocks now respect `delete` flag for removing individual blocks. Previously keys would be deleted only when the entire resource was removed.
  * provider/google: `next_hop_network` on `google_compute_route` is now read-only, to mirror the behavior in the official docs ([#5564](https://github.com/hashicorp/terraform/issues/5564))
  * state/remote/http: PUT requests for this backend will now have `Content-Type: application/json` instead of `application/octet-stream` ([#5499](https://github.com/hashicorp/terraform/issues/5499))

FEATURES:

  * **New command:** `terraform untaint` ([#5527](https://github.com/hashicorp/terraform/issues/5527))
  * **New resource:** `aws_api_gateway_api_key` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_api_gateway_deployment` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_api_gateway_integration_response` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_api_gateway_integration` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_api_gateway_method_response` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_api_gateway_method` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_api_gateway_model` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_api_gateway_resource` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_api_gateway_rest_api` ([#4295](https://github.com/hashicorp/terraform/issues/4295))
  * **New resource:** `aws_elastic_beanstalk_application` ([#3157](https://github.com/hashicorp/terraform/issues/3157))
  * **New resource:** `aws_elastic_beanstalk_configuration_template` ([#3157](https://github.com/hashicorp/terraform/issues/3157))
  * **New resource:** `aws_elastic_beanstalk_environment` ([#3157](https://github.com/hashicorp/terraform/issues/3157))
  * **New resource:** `aws_iam_account_password_policy` ([#5029](https://github.com/hashicorp/terraform/issues/5029))
  * **New resource:** `aws_kms_alias` ([#3928](https://github.com/hashicorp/terraform/issues/3928))
  * **New resource:** `aws_kms_key` ([#3928](https://github.com/hashicorp/terraform/issues/3928))
  * **New resource:** `google_compute_instance_group` ([#4087](https://github.com/hashicorp/terraform/issues/4087))

IMPROVEMENTS:

  * provider/aws: Add `repository_link` as a computed field for `aws_ecr_repository` ([#5524](https://github.com/hashicorp/terraform/issues/5524))
  * provider/aws: Add ability to update Route53 zone comments ([#5318](https://github.com/hashicorp/terraform/issues/5318))
  * provider/aws: Add support for Metrics Collection to `aws_autoscaling_group` ([#4688](https://github.com/hashicorp/terraform/issues/4688))
  * provider/aws: Add support for `description` to `aws_network_interface` ([#5523](https://github.com/hashicorp/terraform/issues/5523))
  * provider/aws: Add support for `storage_encrypted` to `aws_rds_cluster` ([#5520](https://github.com/hashicorp/terraform/issues/5520))
  * provider/aws: Add support for routing rules on `aws_s3_bucket` resources ([#5327](https://github.com/hashicorp/terraform/issues/5327))
  * provider/aws: Enable updates & versioning for `aws_s3_bucket_object` ([#5305](https://github.com/hashicorp/terraform/issues/5305))
  * provider/aws: Guard against Nil Reference in Redshift Endpoints ([#5593](https://github.com/hashicorp/terraform/issues/5593))
  * provider/aws: Lambda S3 object version defaults to `$LATEST` if unspecified ([#5370](https://github.com/hashicorp/terraform/issues/5370))
  * provider/aws: Retry DB Creation on IAM propigation error ([#5515](https://github.com/hashicorp/terraform/issues/5515))
  * provider/aws: Support KMS encryption of S3 objects ([#5453](https://github.com/hashicorp/terraform/issues/5453))
  * provider/aws: `aws_autoscaling_lifecycle_hook` now have `notification_target_arn` and `role_arn` as optional ([#5616](https://github.com/hashicorp/terraform/issues/5616))
  * provider/aws: `aws_ecs_service` validates number of `load_balancer`s before creation/updates ([#5605](https://github.com/hashicorp/terraform/issues/5605))
  * provider/aws: send Terraform version in User-Agent ([#5621](https://github.com/hashicorp/terraform/issues/5621))
  * provider/cloudflare: Change `cloudflare_record` type to ForceNew ([#5353](https://github.com/hashicorp/terraform/issues/5353))
  * provider/consul: `consul_keys` now detects drift and supports deletion of individual `key` blocks ([#5210](https://github.com/hashicorp/terraform/issues/5210))
  * provider/digitalocean: Guard against Nil reference in `digitalocean_droplet` ([#5588](https://github.com/hashicorp/terraform/issues/5588))
  * provider/docker: Add support for `unless-stopped` to docker container `restart_policy` ([#5337](https://github.com/hashicorp/terraform/issues/5337))
  * provider/google: Mark `next_hop_network` as read-only on `google_compute_route` ([#5564](https://github.com/hashicorp/terraform/issues/5564))
  * provider/google: Validate VPN tunnel peer_ip at plan time ([#5501](https://github.com/hashicorp/terraform/issues/5501))
  * provider/openstack: Add Support for Domain ID and Domain Name environment variables ([#5355](https://github.com/hashicorp/terraform/issues/5355))
  * provider/openstack: Add support for instances to have multiple ephemeral disks. ([#5131](https://github.com/hashicorp/terraform/issues/5131))
  * provider/openstack: Re-Add server.AccessIPv4 and server.AccessIPv6 ([#5366](https://github.com/hashicorp/terraform/issues/5366))
  * provider/vsphere: Add support for disk init types ([#4284](https://github.com/hashicorp/terraform/issues/4284))
  * provisioner/remote-exec: Clear out scripts after uploading ([#5577](https://github.com/hashicorp/terraform/issues/5577))
  * state/remote/http: Change content type of PUT requests to the more appropriate `application/json` ([#5499](https://github.com/hashicorp/terraform/issues/5499))

BUG FIXES:

  * core: Disallow negative indices in the element() interpolation function, preventing crash ([#5263](https://github.com/hashicorp/terraform/issues/5263))
  * core: Fix issue that caused tainted resource destroys to be improperly filtered out when using -target and a plan file ([#5516](https://github.com/hashicorp/terraform/issues/5516))
  * core: Fix several issues with retry logic causing spurious "timeout while waiting for state to become ..." errors and unnecessary retry loops ([#5460](https://github.com/hashicorp/terraform/issues/5460)), ([#5538](https://github.com/hashicorp/terraform/issues/5538)), ([#5543](https://github.com/hashicorp/terraform/issues/5543)), ([#5553](https://github.com/hashicorp/terraform/issues/5553))
  * core: Includes upstream HCL fix to properly detect unbalanced braces and throw an error ([#5400](https://github.com/hashicorp/terraform/issues/5400))
  * provider/aws: Allow recovering from failed CloudWatch Event Target creation ([#5395](https://github.com/hashicorp/terraform/issues/5395))
  * provider/aws: Fix EC2 Classic SG Rule issue when referencing rules by name ([#5533](https://github.com/hashicorp/terraform/issues/5533))
  * provider/aws: Fix `aws_cloudformation_stack` update for `parameters` & `capabilities` if unmodified ([#5603](https://github.com/hashicorp/terraform/issues/5603))
  * provider/aws: Fix a bug where AWS Kinesis Stream includes closed shards in the shard_count ([#5401](https://github.com/hashicorp/terraform/issues/5401))
  * provider/aws: Fix a bug where ElasticSearch Domain tags were not being set correctly ([#5361](https://github.com/hashicorp/terraform/issues/5361))
  * provider/aws: Fix a bug where `aws_route` would show continual changes in the plan when not computed ([#5321](https://github.com/hashicorp/terraform/issues/5321))
  * provider/aws: Fix a bug where `publicly_assessible` wasn't being set to state in `aws_db_instance` ([#5535](https://github.com/hashicorp/terraform/issues/5535))
  * provider/aws: Fix a bug where listener protocol on `aws_elb` resources was case insensitive ([#5376](https://github.com/hashicorp/terraform/issues/5376))
  * provider/aws: Fix a bug which caused panics creating rules on security groups in EC2 Classic ([#5329](https://github.com/hashicorp/terraform/issues/5329))
  * provider/aws: Fix crash when `aws_lambda_function` VpcId is nil ([#5182](https://github.com/hashicorp/terraform/issues/5182))
  * provider/aws: Fix error with parsing JSON in `aws_s3_bucket` policy attribute ([#5474](https://github.com/hashicorp/terraform/issues/5474))
  * provider/aws: `aws_lambda_function` can be properly updated, either via `s3_object_version` or via `filename` & `source_code_hash` as described in docs ([#5239](https://github.com/hashicorp/terraform/issues/5239))
  * provider/google: Fix managed instance group preemptible instance creation ([#4834](https://github.com/hashicorp/terraform/issues/4834))
  * provider/openstack: Account for a 403 reply when os-tenant-networks is disabled ([#5432](https://github.com/hashicorp/terraform/issues/5432))
  * provider/openstack: Fix crashing during certain network updates in instances ([#5365](https://github.com/hashicorp/terraform/issues/5365))
  * provider/openstack: Fix create/delete statuses in load balancing resources ([#5557](https://github.com/hashicorp/terraform/issues/5557))
  * provider/openstack: Fix race condition between instance deletion and volume detachment ([#5359](https://github.com/hashicorp/terraform/issues/5359))
  * provider/template: Warn when `template` attribute specified as path ([#5563](https://github.com/hashicorp/terraform/issues/5563))

INTERNAL IMPROVEMENTS:

  * helper/schema: `MaxItems` attribute on schema lists and sets ([#5218](https://github.com/hashicorp/terraform/issues/5218))

## 0.6.12 (February 24, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

  * The `publicly_accessible` attribute on `aws_redshift_cluster` resources now defaults to true

FEATURES:

  * **New command:** `validate` to perform syntax validation ([#3783](https://github.com/hashicorp/terraform/issues/3783))
  * **New provider:** `datadog` ([#5251](https://github.com/hashicorp/terraform/issues/5251))
  * **New interpolation function:** `md5` ([#5267](https://github.com/hashicorp/terraform/issues/5267))
  * **New interpolation function:** `signum` ([#4854](https://github.com/hashicorp/terraform/issues/4854))
  * **New resource:** `aws_cloudwatch_event_rule` ([#4986](https://github.com/hashicorp/terraform/issues/4986))
  * **New resource:** `aws_cloudwatch_event_target` ([#4986](https://github.com/hashicorp/terraform/issues/4986))
  * **New resource:** `aws_lambda_permission` ([#4826](https://github.com/hashicorp/terraform/issues/4826))
  * **New resource:** `azurerm_dns_a_record` ([#5013](https://github.com/hashicorp/terraform/issues/5013))
  * **New resource:** `azurerm_dns_aaaa_record` ([#5013](https://github.com/hashicorp/terraform/issues/5013))
  * **New resource:** `azurerm_dns_cname_record` ([#5013](https://github.com/hashicorp/terraform/issues/5013))
  * **New resource:** `azurerm_dns_mx_record` ([#5041](https://github.com/hashicorp/terraform/issues/5041))
  * **New resource:** `azurerm_dns_ns_record` ([#5041](https://github.com/hashicorp/terraform/issues/5041))
  * **New resource:** `azurerm_dns_srv_record` ([#5041](https://github.com/hashicorp/terraform/issues/5041))
  * **New resource:** `azurerm_dns_txt_record` ([#5041](https://github.com/hashicorp/terraform/issues/5041))
  * **New resource:** `azurerm_dns_zone` ([#4979](https://github.com/hashicorp/terraform/issues/4979))
  * **New resource:** `azurerm_search_service` ([#5203](https://github.com/hashicorp/terraform/issues/5203))
  * **New resource:** `azurerm_sql_database` ([#5003](https://github.com/hashicorp/terraform/issues/5003))
  * **New resource:** `azurerm_sql_firewall_rule` ([#5057](https://github.com/hashicorp/terraform/issues/5057))
  * **New resource:** `azurerm_sql_server` ([#4991](https://github.com/hashicorp/terraform/issues/4991))
  * **New resource:** `google_compute_subnetwork` ([#5130](https://github.com/hashicorp/terraform/issues/5130))

IMPROVEMENTS:

  * core: Backend names are now down cased during `init` in the same manner as `remote config` ([#5012](https://github.com/hashicorp/terraform/issues/5012))
  * core: Upgrade resource name validation warning to an error as planned ([#5272](https://github.com/hashicorp/terraform/issues/5272))
  * core: output "diffs didn't match" error details ([#5276](https://github.com/hashicorp/terraform/issues/5276))
  * provider/aws: Add `is_multi_region_trail` option to CloudTrail ([#4939](https://github.com/hashicorp/terraform/issues/4939))
  * provider/aws: Add support for HTTP(S) endpoints that auto confirm SNS subscription ([#4711](https://github.com/hashicorp/terraform/issues/4711))
  * provider/aws: Add support for Tags to CloudTrail ([#5135](https://github.com/hashicorp/terraform/issues/5135))
  * provider/aws: Add support for Tags to ElasticSearch ([#4973](https://github.com/hashicorp/terraform/issues/4973))
  * provider/aws: Add support for deployment configuration to `aws_ecs_service` ([#5220](https://github.com/hashicorp/terraform/issues/5220))
  * provider/aws: Add support for log validation + KMS encryption to `aws_cloudtrail` ([#5051](https://github.com/hashicorp/terraform/issues/5051))
  * provider/aws: Allow name-prefix and auto-generated names for IAM Server Cert ([#5178](https://github.com/hashicorp/terraform/issues/5178))
  * provider/aws: Expose additional VPN Connection attributes ([#5032](https://github.com/hashicorp/terraform/issues/5032))
  * provider/aws: Return an error if no matching route is found for an AWS Route ([#5155](https://github.com/hashicorp/terraform/issues/5155))
  * provider/aws: Support custom endpoints for AWS EC2 ELB and IAM ([#5114](https://github.com/hashicorp/terraform/issues/5114))
  * provider/aws: The `cluster_type` on `aws_redshift_cluster` resources is now computed ([#5238](https://github.com/hashicorp/terraform/issues/5238))
  * provider/aws: `aws_lambda_function` resources now support VPC configuration ([#5149](https://github.com/hashicorp/terraform/issues/5149))
  * provider/aws: Add support for Enhanced Monitoring to RDS Instances ([#4945](https://github.com/hashicorp/terraform/issues/4945))
  * provider/aws: Improve vpc cidr_block err message ([#5255](https://github.com/hashicorp/terraform/issues/5255))
  * provider/aws: Implement Retention Period for `aws_kinesis_stream` ([#5223](https://github.com/hashicorp/terraform/issues/5223))
  * provider/aws: Enable `stream_arm` output for DynamoDB Table when streams are enabled ([#5271](https://github.com/hashicorp/terraform/issues/5271))
  * provider/digitalocean: `digitalocean_record` resources now export a computed `fqdn` attribute ([#5071](https://github.com/hashicorp/terraform/issues/5071))
  * provider/google: Add assigned IP Address to CloudSQL Instance `google_sql_database_instance` ([#5245](https://github.com/hashicorp/terraform/issues/5245))
  * provider/openstack: Add support for Distributed Routers ([#4878](https://github.com/hashicorp/terraform/issues/4878))
  * provider/openstack: Add support for optional cacert_file parameter ([#5106](https://github.com/hashicorp/terraform/issues/5106))

BUG FIXES:

  * core: Fix bug detecting deeply nested module orphans ([#5022](https://github.com/hashicorp/terraform/issues/5022))
  * core: Fix bug where `ignore_changes` could produce "diffs didn't match during apply" errors ([#4965](https://github.com/hashicorp/terraform/issues/4965))
  * core: Fix race condition when handling tainted resource destroys ([#5026](https://github.com/hashicorp/terraform/issues/5026))
  * core: Improve handling of Provisioners in the graph, fixing "Provisioner already initialized" errors ([#4877](https://github.com/hashicorp/terraform/issues/4877))
  * core: Skip `create_before_destroy` processing during a `terraform destroy`, solving several issues preventing `destroy`
          from working properly with CBD resources ([#5096](https://github.com/hashicorp/terraform/issues/5096))
  * core: Error instead of panic on self var in wrong scope ([#5273](https://github.com/hashicorp/terraform/issues/5273))
  * provider/aws: Fix Copy of Tags to DB Instance when created from Snapshot ([#5197](https://github.com/hashicorp/terraform/issues/5197))
  * provider/aws: Fix DynamoDB Table Refresh to ensure deleted tables are removed from state ([#4943](https://github.com/hashicorp/terraform/issues/4943))
  * provider/aws: Fix ElasticSearch `domain_name` validation ([#4973](https://github.com/hashicorp/terraform/issues/4973))
  * provider/aws: Fix issue applying security group changes in EC2 Classic RDS for aws_db_instance ([#4969](https://github.com/hashicorp/terraform/issues/4969))
  * provider/aws: Fix reading auto scaling group availability zones ([#5044](https://github.com/hashicorp/terraform/issues/5044))
  * provider/aws: Fix reading auto scaling group load balancers ([#5045](https://github.com/hashicorp/terraform/issues/5045))
  * provider/aws: Fix `aws_redshift_cluster` to allow `publicly_accessible` to be false ([#5262](https://github.com/hashicorp/terraform/issues/5262))
  * provider/aws: Wait longer for internet gateways to detach ([#5120](https://github.com/hashicorp/terraform/issues/5120))
  * provider/aws: Fix issue reading auto scaling group termination policies ([#5101](https://github.com/hashicorp/terraform/issues/5101))
  * provider/cloudflare: `ttl` no longer shows a change on each plan on `cloudflare_record` resources ([#5042](https://github.com/hashicorp/terraform/issues/5042))
  * provider/docker: Fix the default docker_host value ([#5088](https://github.com/hashicorp/terraform/issues/5088))
  * provider/google: Fix backend service max_utilization attribute ([#5075](https://github.com/hashicorp/terraform/issues/5075))
  * provider/google: Fix reading of `google_compute_vpn_gateway` without an explicit ([#5125](https://github.com/hashicorp/terraform/issues/5125))
  * provider/google: Fix crash when setting `ack_deadline_seconds` on `google_pubsub_subscription` ([#5110](https://github.com/hashicorp/terraform/issues/5110))
  * provider/openstack: Fix crash when `access_network` was not defined in instances ([#4966](https://github.com/hashicorp/terraform/issues/4966))
  * provider/powerdns: Fix refresh of `powerdns_record` no longer fails if the record name contains a `-` ([#5228](https://github.com/hashicorp/terraform/issues/5228))
  * provider/vcd: Wait for DHCP assignment when creating `vcd_vapp` resources with no static IP assignment ([#5195](https://github.com/hashicorp/terraform/issues/5195))

## 0.6.11 (February 1, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

  * The `max_size`, `min_size` and `desired_capacity` attributes on `aws_autoscaling_schedule` resources now default to 0

FEATURES:

  * **New provider: `powerdns` - PowerDNS REST API** ([#4885](https://github.com/hashicorp/terraform/issues/4885))
  * **New builtin function:** `trimspace` for trimming whitespaces ([#4910](https://github.com/hashicorp/terraform/issues/4910))
  * **New builtin function:** `base64sha256` for base64 encoding raw sha256 sum of a given string ([#4899](https://github.com/hashicorp/terraform/issues/4899))
  * **New resource:** `openstack_lb_member_v1` ([#4359](https://github.com/hashicorp/terraform/issues/4359))

IMPROVEMENTS:

  * provider/template: Remove unnecessary mime-type validation from `template_cloudinit_config` resources ([#4873](https://github.com/hashicorp/terraform/issues/4873))
  * provider/template: Correct spelling of "Boundary" in the part separator of rendered `template_cloudinit_config` resources ([#4873](https://github.com/hashicorp/terraform/issues/4873))
  * provider/aws: Provide a better message if no AWS creds are found ([#4869](https://github.com/hashicorp/terraform/issues/4869))
  * provider/openstack: Ability to specify per-network Floating IPs ([#4812](https://github.com/hashicorp/terraform/issues/4812))

BUG FIXES:

  * provider/aws: `aws_autoscale_schedule` 0 values ([#4693](https://github.com/hashicorp/terraform/issues/4693))
  * provider/aws: Fix regression with VPCs and ClassicLink for regions that do not support it ([#4879](https://github.com/hashicorp/terraform/issues/4879))
  * provider/aws: Change VPC ClassicLink to be computed ([#4933](https://github.com/hashicorp/terraform/issues/4933))
  * provider/aws: Fix SNS Topic Refresh to ensure deleted topics are removed from state ([#4891](https://github.com/hashicorp/terraform/issues/4891))
  * provider/aws: Refactor Route53 record to fix regression in deleting records created in previous versions of Terraform ([#4892](https://github.com/hashicorp/terraform/issues/4892))
  * provider/azurerm: Fix panic if no creds supplied ([#4902](https://github.com/hashicorp/terraform/issues/4902))
  * provider/openstack: Changing the port resource to mark the ip_address as optional ([#4850](https://github.com/hashicorp/terraform/issues/4850))
  * provider/docker: Catch potential custom network errors in docker ([#4918](https://github.com/hashicorp/terraform/issues/4918))



## 0.6.10 (January 27, 2016)

BACKWARDS INCOMPATIBILITIES / NOTES:

  * The `-module-depth` flag available on `plan`, `apply`, `show`, and `graph` now defaults to `-1`, causing
    resources within modules to be expanded in command output. This is only a cosmetic change; it does not affect
    any behavior.
  * This release includes a bugfix for `$${}` interpolation escaping. These strings are now properly converted to `${}`
    during interpolation. This may cause diffs on existing configurations in certain cases.
  * Users of `consul_keys` should note that the `value` sub-attribute of `key` will no longer be updated with the remote value of the key. It should be only used to _set_ a key in Consul K/V. To reference key values, use the `var` attribute.
  * The 0.6.9 release contained a regression in `aws_autoscaling_group` capacity waiting behavior for configs where `min_elb_capacity != desired_capacity` or `min_size != desired_capacity`. This release remedies that regression by un-deprecating `min_elb_capacity` and restoring the prior behavior.
  * Users of `aws_security_group` may notice new diffs in initial plans with 0.6.10 due to a bugfix that fixes drift detection on nested security group rules. These new diffs should reflect the actual state of the resources, which Terraform previously was unable to see.


FEATURES:

  * **New resource: `aws_lambda_alias`** ([#4664](https://github.com/hashicorp/terraform/issues/4664))
  * **New resource: `aws_redshift_cluster`** ([#3862](https://github.com/hashicorp/terraform/issues/3862))
  * **New resource: `aws_redshift_parameter_group`** ([#3862](https://github.com/hashicorp/terraform/issues/3862))
  * **New resource: `aws_redshift_security_group`** ([#3862](https://github.com/hashicorp/terraform/issues/3862))
  * **New resource: `aws_redshift_subnet_group`** ([#3862](https://github.com/hashicorp/terraform/issues/3862))
  * **New resource: `azurerm_cdn_endpoint`** ([#4759](https://github.com/hashicorp/terraform/issues/4759))
  * **New resource: `azurerm_cdn_profile`** ([#4740](https://github.com/hashicorp/terraform/issues/4740))
  * **New resource: `azurerm_network_interface`** ([#4598](https://github.com/hashicorp/terraform/issues/4598))
  * **New resource: `azurerm_network_security_rule`** ([#4586](https://github.com/hashicorp/terraform/issues/4586))
  * **New resource: `azurerm_route_table`** ([#4602](https://github.com/hashicorp/terraform/issues/4602))
  * **New resource: `azurerm_route`** ([#4604](https://github.com/hashicorp/terraform/issues/4604))
  * **New resource: `azurerm_storage_account`** ([#4698](https://github.com/hashicorp/terraform/issues/4698))
  * **New resource: `azurerm_storage_blob`** ([#4862](https://github.com/hashicorp/terraform/issues/4862))
  * **New resource: `azurerm_storage_container`** ([#4862](https://github.com/hashicorp/terraform/issues/4862))
  * **New resource: `azurerm_storage_queue`** ([#4862](https://github.com/hashicorp/terraform/issues/4862))
  * **New resource: `azurerm_subnet`** ([#4595](https://github.com/hashicorp/terraform/issues/4595))
  * **New resource: `docker_network`** ([#4483](https://github.com/hashicorp/terraform/issues/4483))
  * **New resource: `docker_volume`** ([#4483](https://github.com/hashicorp/terraform/issues/4483))
  * **New resource: `google_sql_user`** ([#4669](https://github.com/hashicorp/terraform/issues/4669))

IMPROVEMENTS:

  * core: Add `sha256()` interpolation function ([#4704](https://github.com/hashicorp/terraform/issues/4704))
  * core: Validate lifecycle keys to show helpful error messages whe they are mistypes ([#4745](https://github.com/hashicorp/terraform/issues/4745))
  * core: Default `module-depth` parameter to `-1`, which expands resources within modules in command output ([#4763](https://github.com/hashicorp/terraform/issues/4763))
  * core: Variable types may now be specified explicitly using the `type` argument ([#4795](https://github.com/hashicorp/terraform/issues/4795))
  * provider/aws: Add new parameters `az_mode` and `availability_zone(s)` in ElastiCache ([#4631](https://github.com/hashicorp/terraform/issues/4631))
  * provider/aws: Allow ap-northeast-2 (Seoul) as valid region ([#4637](https://github.com/hashicorp/terraform/issues/4637))
  * provider/aws: Limit SNS Topic Subscription protocols ([#4639](https://github.com/hashicorp/terraform/issues/4639))
  * provider/aws: Add support for configuring logging on `aws_s3_bucket` resources ([#4482](https://github.com/hashicorp/terraform/issues/4482))
  * provider/aws: Add AWS Classiclink for AWS VPC resource ([#3994](https://github.com/hashicorp/terraform/issues/3994))
  * provider/aws: Supporting New AWS Route53 HealthCheck additions ([#4564](https://github.com/hashicorp/terraform/issues/4564))
  * provider/aws: Store instance state ([#3261](https://github.com/hashicorp/terraform/issues/3261))
  * provider/aws: Add support for updating ELB availability zones and subnets ([#4597](https://github.com/hashicorp/terraform/issues/4597))
  * provider/aws: Enable specifying aws s3 redirect protocol ([#4098](https://github.com/hashicorp/terraform/issues/4098))
  * provider/aws: Added support for `encrypted` on `ebs_block_devices` in Launch Configurations ([#4481](https://github.com/hashicorp/terraform/issues/4481))
  * provider/aws: Retry Listener Creation for ELBs ([#4825](https://github.com/hashicorp/terraform/issues/4825))
  * provider/aws: Add support for creating Managed Microsoft Active Directory
    and Directory Connectors ([#4388](https://github.com/hashicorp/terraform/issues/4388))
  * provider/aws: Mark some `aws_db_instance` fields as optional ([#3138](https://github.com/hashicorp/terraform/issues/3138))
  * provider/digitalocean: Add support for reassigning `digitalocean_floating_ip` resources ([#4476](https://github.com/hashicorp/terraform/issues/4476))
  * provider/dme: Add support for Global Traffic Director locations on `dme_record` resources ([#4305](https://github.com/hashicorp/terraform/issues/4305))
  * provider/docker: Add support for adding host entries on `docker_container` resources ([#3463](https://github.com/hashicorp/terraform/issues/3463))
  * provider/docker: Add support for mounting named volumes on `docker_container` resources ([#4480](https://github.com/hashicorp/terraform/issues/4480))
  * provider/google: Add content field to bucket object ([#3893](https://github.com/hashicorp/terraform/issues/3893))
  * provider/google: Add support for  `named_port` blocks on `google_compute_instance_group_manager` resources ([#4605](https://github.com/hashicorp/terraform/issues/4605))
  * provider/openstack: Add "personality" support to instance resource ([#4623](https://github.com/hashicorp/terraform/issues/4623))
  * provider/packet: Handle external state changes for Packet resources gracefully ([#4676](https://github.com/hashicorp/terraform/issues/4676))
  * provider/tls: `tls_private_key` now exports attributes with public key in both PEM and OpenSSH format ([#4606](https://github.com/hashicorp/terraform/issues/4606))
  * provider/vdc: Add `allow_unverified_ssl` for connections to vCloud API ([#4811](https://github.com/hashicorp/terraform/issues/4811))
  * state/remote: Allow KMS Key Encryption to be used with S3 backend ([#2903](https://github.com/hashicorp/terraform/issues/2903))

BUG FIXES:

  * core: Fix handling of literals with escaped interpolations `$${var}` ([#4747](https://github.com/hashicorp/terraform/issues/4747))
  * core: Fix diff mismatch when RequiresNew field and list both change ([#4749](https://github.com/hashicorp/terraform/issues/4749))
  * core: Respect module target path argument on `terraform init` ([#4753](https://github.com/hashicorp/terraform/issues/4753))
  * core: Write planfile even on empty plans ([#4766](https://github.com/hashicorp/terraform/issues/4766))
  * core: Add validation error when output is missing value field ([#4762](https://github.com/hashicorp/terraform/issues/4762))
  * core: Fix improper handling of orphan resources when targeting ([#4574](https://github.com/hashicorp/terraform/issues/4574))
  * core: Properly handle references to computed set attributes ([#4840](https://github.com/hashicorp/terraform/issues/4840))
  * config: Detect a specific JSON edge case and show a helpful workaround ([#4746](https://github.com/hashicorp/terraform/issues/4746))
  * provider/openstack: Ensure valid Security Group Rule attribute combination ([#4466](https://github.com/hashicorp/terraform/issues/4466))
  * provider/openstack: Don't put fixed_ip in port creation request if not defined ([#4617](https://github.com/hashicorp/terraform/issues/4617))
  * provider/google: Clarify SQL Database Instance recent name restriction ([#4577](https://github.com/hashicorp/terraform/issues/4577))
  * provider/google: Split Instance network interface into two fields ([#4265](https://github.com/hashicorp/terraform/issues/4265))
  * provider/aws: Error with empty list item on security group ([#4140](https://github.com/hashicorp/terraform/issues/4140))
  * provider/aws: Fix issue with detecting drift in AWS Security Groups rules ([#4779](https://github.com/hashicorp/terraform/issues/4779))
  * provider/aws: Trap Instance error from mismatched SG IDs and Names ([#4240](https://github.com/hashicorp/terraform/issues/4240))
  * provider/aws: EBS optimised to force new resource in AWS Instance ([#4627](https://github.com/hashicorp/terraform/issues/4627))
  * provider/aws: Wait for NACL rule to be visible ([#4734](https://github.com/hashicorp/terraform/issues/4734))
  * provider/aws: `default_result` on `aws_autoscaling_lifecycle_hook` resources is now computed ([#4695](https://github.com/hashicorp/terraform/issues/4695))
  * provider/aws: fix ASG capacity waiting regression by un-deprecating `min_elb_capacity` ([#4864](https://github.com/hashicorp/terraform/issues/4864))
  * provider/consul: fix several bugs surrounding update behavior ([#4787](https://github.com/hashicorp/terraform/issues/4787))
  * provider/mailgun: Handle the fact that the domain destroy API is eventually consistent ([#4777](https://github.com/hashicorp/terraform/issues/4777))
  * provider/template: Fix race causing sporadic crashes in template_file with count > 1 ([#4694](https://github.com/hashicorp/terraform/issues/4694))
  * provider/template: Add support for updating `template_cloudinit_config` resources ([#4757](https://github.com/hashicorp/terraform/issues/4757))
  * provisioner/chef: Add ENV['no_proxy'] to chef provisioner if no_proxy is detected ([#4661](https://github.com/hashicorp/terraform/issues/4661))

## 0.6.9 (January 8, 2016)

FEATURES:

  * **New provider: `vcd` - VMware vCloud Director** ([#3785](https://github.com/hashicorp/terraform/issues/3785))
  * **New provider: `postgresql` - Create PostgreSQL databases and roles** ([#3653](https://github.com/hashicorp/terraform/issues/3653))
  * **New provider: `chef` - Create chef environments, roles, etc** ([#3084](https://github.com/hashicorp/terraform/issues/3084))
  * **New provider: `azurerm` - Preliminary support for Azure Resource Manager** ([#4226](https://github.com/hashicorp/terraform/issues/4226))
  * **New provider: `mysql` - Create MySQL databases** ([#3122](https://github.com/hashicorp/terraform/issues/3122))
  * **New resource: `aws_autoscaling_schedule`** ([#4256](https://github.com/hashicorp/terraform/issues/4256))
  * **New resource: `aws_nat_gateway`** ([#4381](https://github.com/hashicorp/terraform/issues/4381))
  * **New resource: `aws_network_acl_rule`** ([#4286](https://github.com/hashicorp/terraform/issues/4286))
  * **New resources: `aws_ecr_repository` and `aws_ecr_repository_policy`** ([#4415](https://github.com/hashicorp/terraform/issues/4415))
  * **New resource: `google_pubsub_topic`** ([#3671](https://github.com/hashicorp/terraform/issues/3671))
  * **New resource: `google_pubsub_subscription`** ([#3671](https://github.com/hashicorp/terraform/issues/3671))
  * **New resource: `template_cloudinit_config`** ([#4095](https://github.com/hashicorp/terraform/issues/4095))
  * **New resource: `tls_locally_signed_cert`** ([#3930](https://github.com/hashicorp/terraform/issues/3930))
  * **New remote state backend: `artifactory`** ([#3684](https://github.com/hashicorp/terraform/issues/3684))

IMPROVEMENTS:

  * core: Change set internals for performance improvements ([#3992](https://github.com/hashicorp/terraform/issues/3992))
  * core: Support HTTP basic auth in consul remote state ([#4166](https://github.com/hashicorp/terraform/issues/4166))
  * core: Improve error message on resource arity mismatch ([#4244](https://github.com/hashicorp/terraform/issues/4244))
  * core: Add support for unary operators + and - to the interpolation syntax ([#3621](https://github.com/hashicorp/terraform/issues/3621))
  * core: Add SSH agent support for Windows ([#4323](https://github.com/hashicorp/terraform/issues/4323))
  * core: Add `sha1()` interpolation function ([#4450](https://github.com/hashicorp/terraform/issues/4450))
  * provider/aws: Add `placement_group` as an option for `aws_autoscaling_group` ([#3704](https://github.com/hashicorp/terraform/issues/3704))
  * provider/aws: Add support for DynamoDB Table StreamSpecifications ([#4208](https://github.com/hashicorp/terraform/issues/4208))
  * provider/aws: Add `name_prefix` to Security Groups ([#4167](https://github.com/hashicorp/terraform/issues/4167))
  * provider/aws: Add support for removing nodes to `aws_elasticache_cluster` ([#3809](https://github.com/hashicorp/terraform/issues/3809))
  * provider/aws: Add support for `skip_final_snapshot` to `aws_db_instance` ([#3853](https://github.com/hashicorp/terraform/issues/3853))
  * provider/aws: Adding support for Tags to DB SecurityGroup ([#4260](https://github.com/hashicorp/terraform/issues/4260))
  * provider/aws: Adding Tag support for DB Param Groups ([#4259](https://github.com/hashicorp/terraform/issues/4259))
  * provider/aws: Fix issue with updated route ids for VPC Endpoints ([#4264](https://github.com/hashicorp/terraform/issues/4264))
  * provider/aws: Added measure_latency option to Route 53 Health Check resource ([#3688](https://github.com/hashicorp/terraform/issues/3688))
  * provider/aws: Validate IOPs for EBS Volumes ([#4146](https://github.com/hashicorp/terraform/issues/4146))
  * provider/aws: DB Subnet group arn output ([#4261](https://github.com/hashicorp/terraform/issues/4261))
  * provider/aws: Get full Kinesis streams view with pagination ([#4368](https://github.com/hashicorp/terraform/issues/4368))
  * provider/aws: Allow changing private IPs for ENIs ([#4307](https://github.com/hashicorp/terraform/issues/4307))
  * provider/aws: Retry MalformedPolicy errors due to newly created principals in S3 Buckets ([#4315](https://github.com/hashicorp/terraform/issues/4315))
  * provider/aws: Validate `name` on `db_subnet_group` against AWS requirements ([#4340](https://github.com/hashicorp/terraform/issues/4340))
  * provider/aws: wait for ASG capacity on update ([#3947](https://github.com/hashicorp/terraform/issues/3947))
  * provider/aws: Add validation for ECR repository name ([#4431](https://github.com/hashicorp/terraform/issues/4431))
  * provider/cloudstack: performance improvements ([#4150](https://github.com/hashicorp/terraform/issues/4150))
  * provider/docker: Add support for setting the entry point on `docker_container` resources ([#3761](https://github.com/hashicorp/terraform/issues/3761))
  * provider/docker: Add support for setting the restart policy on `docker_container` resources ([#3761](https://github.com/hashicorp/terraform/issues/3761))
  * provider/docker: Add support for setting memory, swap and CPU shares on `docker_container` resources ([#3761](https://github.com/hashicorp/terraform/issues/3761))
  * provider/docker: Add support for setting labels on `docker_container` resources ([#3761](https://github.com/hashicorp/terraform/issues/3761))
  * provider/docker: Add support for setting log driver and options on `docker_container` resources ([#3761](https://github.com/hashicorp/terraform/issues/3761))
  * provider/docker: Add support for settings network mode on `docker_container` resources ([#4475](https://github.com/hashicorp/terraform/issues/4475))
  * provider/heroku: Improve handling of Applications within an Organization ([#4495](https://github.com/hashicorp/terraform/issues/4495))
  * provider/vsphere: Add support for custom vm params on `vsphere_virtual_machine` ([#3867](https://github.com/hashicorp/terraform/issues/3867))
  * provider/vsphere: Rename vcenter_server config parameter to something clearer ([#3718](https://github.com/hashicorp/terraform/issues/3718))
  * provider/vsphere: Make allow_unverified_ssl a configuable on the provider ([#3933](https://github.com/hashicorp/terraform/issues/3933))
  * provider/vsphere: Add folder handling for folder-qualified vm names ([#3939](https://github.com/hashicorp/terraform/issues/3939))
  * provider/vsphere: Change ip_address parameter for ipv6 support ([#4035](https://github.com/hashicorp/terraform/issues/4035))
  * provider/openstack: Increase instance timeout from 10 to 30 minutes ([#4223](https://github.com/hashicorp/terraform/issues/4223))
  * provider/google: Add `restart_policy` attribute to `google_managed_instance_group` ([#3892](https://github.com/hashicorp/terraform/issues/3892))

BUG FIXES:

  * core: skip provider input for deprecated fields ([#4193](https://github.com/hashicorp/terraform/issues/4193))
  * core: Fix issue which could cause fields that become empty to retain old values in the state ([#3257](https://github.com/hashicorp/terraform/issues/3257))
  * provider/docker: Fix an issue running with Docker Swarm by looking up containers by ID instead of name ([#4148](https://github.com/hashicorp/terraform/issues/4148))
  * provider/openstack: Better handling of load balancing resource state changes ([#3926](https://github.com/hashicorp/terraform/issues/3926))
  * provider/aws: Treat `INACTIVE` ECS cluster as deleted ([#4364](https://github.com/hashicorp/terraform/issues/4364))
  * provider/aws: Skip `source_security_group_id` determination logic for Classic ELBs ([#4075](https://github.com/hashicorp/terraform/issues/4075))
  * provider/aws: Fix issue destroy Route 53 zone/record if it no longer exists ([#4198](https://github.com/hashicorp/terraform/issues/4198))
  * provider/aws: Fix issue force destroying a versioned S3 bucket ([#4168](https://github.com/hashicorp/terraform/issues/4168))
  * provider/aws: Update DB Replica to honor storage type ([#4155](https://github.com/hashicorp/terraform/issues/4155))
  * provider/aws: Fix issue creating AWS RDS replicas across regions ([#4215](https://github.com/hashicorp/terraform/issues/4215))
  * provider/aws: Fix issue with Route53 and zero weighted records ([#4427](https://github.com/hashicorp/terraform/issues/4427))
  * provider/aws: Fix issue with iam_profile in aws_instance when a path is specified ([#3663](https://github.com/hashicorp/terraform/issues/3663))
  * provider/aws: Refactor AWS Authentication chain to fix issue with authentication and IAM ([#4254](https://github.com/hashicorp/terraform/issues/4254))
  * provider/aws: Fix issue with finding S3 Hosted Zone ID for eu-central-1 region ([#4236](https://github.com/hashicorp/terraform/issues/4236))
  * provider/aws: Fix missing AMI issue with Launch Configurations ([#4242](https://github.com/hashicorp/terraform/issues/4242))
  * provider/aws: Opsworks stack SSH key is write-only ([#4241](https://github.com/hashicorp/terraform/issues/4241))
  * provider/aws: Update VPC Endpoint to correctly set route table ids ([#4392](https://github.com/hashicorp/terraform/issues/4392))
  * provider/aws: Fix issue with ElasticSearch Domain `access_policies` always appear changed ([#4245](https://github.com/hashicorp/terraform/issues/4245))
  * provider/aws: Fix issue with nil parameter group value causing panic in `aws_db_parameter_group` ([#4318](https://github.com/hashicorp/terraform/issues/4318))
  * provider/aws: Fix issue with Elastic IPs not recognizing when they have been unassigned manually ([#4387](https://github.com/hashicorp/terraform/issues/4387))
  * provider/aws: Use body or URL for all CloudFormation stack updates ([#4370](https://github.com/hashicorp/terraform/issues/4370))
  * provider/aws: Fix template_url/template_body conflict ([#4540](https://github.com/hashicorp/terraform/issues/4540))
  * provider/aws: Fix bug w/ changing ECS svc/ELB association ([#4366](https://github.com/hashicorp/terraform/issues/4366))
  * provider/aws: Fix RDS unexpected state config ([#4490](https://github.com/hashicorp/terraform/issues/4490))
  * provider/digitalocean: Fix issue where a floating IP attached to a missing droplet causes a panic ([#4214](https://github.com/hashicorp/terraform/issues/4214))
  * provider/google: Fix project metadata sshKeys from showing up and causing unnecessary diffs ([#4512](https://github.com/hashicorp/terraform/issues/4512))
  * provider/heroku: Retry drain create until log channel is assigned ([#4823](https://github.com/hashicorp/terraform/issues/4823))
  * provider/openstack: Handle volumes in "deleting" state ([#4204](https://github.com/hashicorp/terraform/issues/4204))
  * provider/rundeck: Tolerate Rundeck server not returning project name when reading a job ([#4301](https://github.com/hashicorp/terraform/issues/4301))
  * provider/vsphere: Create and attach additional disks before bootup ([#4196](https://github.com/hashicorp/terraform/issues/4196))
  * provider/openstack: Convert block_device from a Set to a List ([#4288](https://github.com/hashicorp/terraform/issues/4288))
  * provider/google: Terraform identifies deleted resources and handles them appropriately on Read ([#3913](https://github.com/hashicorp/terraform/issues/3913))

## 0.6.8 (December 2, 2015)

FEATURES:

  * **New provider: `statuscake`** ([#3340](https://github.com/hashicorp/terraform/issues/3340))
  * **New resource: `digitalocean_floating_ip`** ([#3748](https://github.com/hashicorp/terraform/issues/3748))
  * **New resource: `aws_lambda_event_source_mapping`** ([#4093](https://github.com/hashicorp/terraform/issues/4093))

IMPROVEMENTS:

  * provider/cloudstack: Reduce the number of network calls required for common operations ([#4051](https://github.com/hashicorp/terraform/issues/4051))
  * provider/aws: Make `publically_accessible` on an `aws_db_instance` update existing instances instead of forcing new ones ([#3895](https://github.com/hashicorp/terraform/issues/3895))
  * provider/aws: Allow `block_duration_minutes` to be set for spot instance requests ([#4071](https://github.com/hashicorp/terraform/issues/4071))
  * provider/aws: Make setting `acl` on S3 buckets update existing buckets instead of forcing new ones ([#4080](https://github.com/hashicorp/terraform/issues/4080))
  * provider/aws: Make updates to `assume_role_policy` modify existing IAM roles instead of forcing new ones ([#4107](https://github.com/hashicorp/terraform/issues/4107))

BUG FIXES:

  * core: Fix a bug which prevented HEREDOC syntax being used in lists ([#4078](https://github.com/hashicorp/terraform/issues/4078))
  * core: Fix a bug which prevented HEREDOC syntax where the anchor ends in a number ([#4128](https://github.com/hashicorp/terraform/issues/4128))
  * core: Fix a bug which prevented HEREDOC syntax being used with Windows line endings ([#4069](https://github.com/hashicorp/terraform/issues/4069))
  * provider/aws: Fix a bug which could result in a panic when reading EC2 metadata ([#4024](https://github.com/hashicorp/terraform/issues/4024))
  * provider/aws: Fix issue recreating security group rule if it has been destroyed ([#4050](https://github.com/hashicorp/terraform/issues/4050))
  * provider/aws: Fix issue with some attributes in Spot Instance Requests returning as nil ([#4132](https://github.com/hashicorp/terraform/issues/4132))
  * provider/aws: Fix issue where SPF records in Route 53 could show differences with no modification to the configuration ([#4108](https://github.com/hashicorp/terraform/issues/4108))
  * provisioner/chef: Fix issue with path separators breaking the Chef provisioner on Windows ([#4041](https://github.com/hashicorp/terraform/issues/4041))

## 0.6.7 (November 23, 2015)

FEATURES:

  * **New provider: `tls`** - A utility provider for generating TLS keys/self-signed certificates for development and testing ([#2778](https://github.com/hashicorp/terraform/issues/2778))
  * **New provider: `dyn`** - Manage DNS records on Dyn
  * **New resource: `aws_cloudformation_stack`** ([#2636](https://github.com/hashicorp/terraform/issues/2636))
  * **New resource: `aws_cloudtrail`** ([#3094](https://github.com/hashicorp/terraform/issues/3094)), ([#4010](https://github.com/hashicorp/terraform/issues/4010))
  * **New resource: `aws_route`** ([#3548](https://github.com/hashicorp/terraform/issues/3548))
  * **New resource: `aws_codecommit_repository`** ([#3274](https://github.com/hashicorp/terraform/issues/3274))
  * **New resource: `aws_kinesis_firehose_delivery_stream`** ([#3833](https://github.com/hashicorp/terraform/issues/3833))
  * **New resource: `google_sql_database` and `google_sql_database_instance`** ([#3617](https://github.com/hashicorp/terraform/issues/3617))
  * **New resource: `google_compute_global_address`** ([#3701](https://github.com/hashicorp/terraform/issues/3701))
  * **New resource: `google_compute_https_health_check`** ([#3883](https://github.com/hashicorp/terraform/issues/3883))
  * **New resource: `google_compute_ssl_certificate`** ([#3723](https://github.com/hashicorp/terraform/issues/3723))
  * **New resource: `google_compute_url_map`** ([#3722](https://github.com/hashicorp/terraform/issues/3722))
  * **New resource: `google_compute_target_http_proxy`** ([#3727](https://github.com/hashicorp/terraform/issues/3727))
  * **New resource: `google_compute_target_https_proxy`** ([#3728](https://github.com/hashicorp/terraform/issues/3728))
  * **New resource: `google_compute_global_forwarding_rule`** ([#3702](https://github.com/hashicorp/terraform/issues/3702))
  * **New resource: `openstack_networking_port_v2`** ([#3731](https://github.com/hashicorp/terraform/issues/3731))
  * New interpolation function: `coalesce` ([#3814](https://github.com/hashicorp/terraform/issues/3814))

IMPROVEMENTS:

  * core: Improve message to list only resources which will be destroyed when using `--target` ([#3859](https://github.com/hashicorp/terraform/issues/3859))
  * connection/ssh: Accept `private_key` contents instead of paths ([#3846](https://github.com/hashicorp/terraform/issues/3846))
  * provider/google: `preemptible` option for instance_template ([#3667](https://github.com/hashicorp/terraform/issues/3667))
  * provider/google: Accurate Terraform Version ([#3554](https://github.com/hashicorp/terraform/issues/3554))
  * provider/google: Simplified auth (DefaultClient support) ([#3553](https://github.com/hashicorp/terraform/issues/3553))
  * provider/google: `automatic_restart`, `preemptible`, `on_host_maintenance` options ([#3643](https://github.com/hashicorp/terraform/issues/3643))
  * provider/google: Read credentials as contents instead of path ([#3901](https://github.com/hashicorp/terraform/issues/3901))
  * null_resource: Enhance and document ([#3244](https://github.com/hashicorp/terraform/issues/3244), [#3659](https://github.com/hashicorp/terraform/issues/3659))
  * provider/aws: Add CORS settings to S3 bucket ([#3387](https://github.com/hashicorp/terraform/issues/3387))
  * provider/aws: Add notification topic ARN for ElastiCache clusters ([#3674](https://github.com/hashicorp/terraform/issues/3674))
  * provider/aws: Add `kinesis_endpoint` for configuring Kinesis ([#3255](https://github.com/hashicorp/terraform/issues/3255))
  * provider/aws: Add a computed ARN for S3 Buckets ([#3685](https://github.com/hashicorp/terraform/issues/3685))
  * provider/aws: Add S3 support for Lambda Function resource ([#3794](https://github.com/hashicorp/terraform/issues/3794))
  * provider/aws: Add `name_prefix` option to launch configurations ([#3802](https://github.com/hashicorp/terraform/issues/3802))
  * provider/aws: Add support for group name and path changes with IAM group update function ([#3237](https://github.com/hashicorp/terraform/issues/3237))
  * provider/aws: Provide `source_security_group_id` for ELBs inside a VPC ([#3780](https://github.com/hashicorp/terraform/issues/3780))
  * provider/aws: Add snapshot window and retention limits for ElastiCache (Redis) ([#3707](https://github.com/hashicorp/terraform/issues/3707))
  * provider/aws: Add username updates for `aws_iam_user` ([#3227](https://github.com/hashicorp/terraform/issues/3227))
  * provider/aws: Add AutoMinorVersionUpgrade to RDS Instances ([#3677](https://github.com/hashicorp/terraform/issues/3677))
  * provider/aws: Add `access_logs` to ELB resource ([#3756](https://github.com/hashicorp/terraform/issues/3756))
  * provider/aws: Add a retry function to rescue an error in creating Autoscaling Lifecycle Hooks ([#3694](https://github.com/hashicorp/terraform/issues/3694))
  * provider/aws: `engine_version` is now optional for DB Instance ([#3744](https://github.com/hashicorp/terraform/issues/3744))
  * provider/aws: Add configuration to enable copying RDS tags to final snapshot ([#3529](https://github.com/hashicorp/terraform/issues/3529))
  * provider/aws: RDS Cluster additions (`backup_retention_period`, `preferred_backup_window`, `preferred_maintenance_window`) ([#3757](https://github.com/hashicorp/terraform/issues/3757))
  * provider/aws: Document and validate ELB `ssl_certificate_id` and protocol requirements ([#3887](https://github.com/hashicorp/terraform/issues/3887))
  * provider/azure: Read `publish_settings` as contents instead of path ([#3899](https://github.com/hashicorp/terraform/issues/3899))
  * provider/openstack: Use IPv4 as the default IP version for subnets ([#3091](https://github.com/hashicorp/terraform/issues/3091))
  * provider/aws: Apply security group after restoring `db_instance` from snapshot ([#3513](https://github.com/hashicorp/terraform/issues/3513))
  * provider/aws: Make the AutoScalingGroup `name` optional ([#3710](https://github.com/hashicorp/terraform/issues/3710))
  * provider/openstack: Add "delete on termination" boot-from-volume option ([#3232](https://github.com/hashicorp/terraform/issues/3232))
  * provider/digitalocean: Make `user_data` force a new droplet ([#3740](https://github.com/hashicorp/terraform/issues/3740))
  * provider/vsphere: Do not add network interfaces by default ([#3652](https://github.com/hashicorp/terraform/issues/3652))
  * provider/openstack: Configure Fixed IPs through ports ([#3772](https://github.com/hashicorp/terraform/issues/3772))
  * provider/openstack: Specify a port ID on a Router Interface ([#3903](https://github.com/hashicorp/terraform/issues/3903))
  * provider/openstack: Make LBaaS Virtual IP computed ([#3927](https://github.com/hashicorp/terraform/issues/3927))

BUG FIXES:

  * `terraform remote config`: update `--help` output ([#3632](https://github.com/hashicorp/terraform/issues/3632))
  * core: Modules on Git branches now update properly ([#1568](https://github.com/hashicorp/terraform/issues/1568))
  * core: Fix issue preventing input prompts for unset variables during plan ([#3843](https://github.com/hashicorp/terraform/issues/3843))
  * core: Fix issue preventing input prompts for unset variables during refresh ([#4017](https://github.com/hashicorp/terraform/issues/4017))
  * core: Orphan resources can now be targets ([#3912](https://github.com/hashicorp/terraform/issues/3912))
  * helper/schema: Skip StateFunc when value is nil ([#4002](https://github.com/hashicorp/terraform/issues/4002))
  * provider/google: Timeout when deleting large `instance_group_manager` ([#3591](https://github.com/hashicorp/terraform/issues/3591))
  * provider/aws: Fix issue with order of Termination Policies in AutoScaling Groups.
      This will introduce plans on upgrade to this version, in order to correct the ordering ([#2890](https://github.com/hashicorp/terraform/issues/2890))
  * provider/aws: Allow cluster name, not only ARN for `aws_ecs_service` ([#3668](https://github.com/hashicorp/terraform/issues/3668))
  * provider/aws: Fix a bug where a non-lower-cased `maintenance_window` can cause unnecessary planned changes ([#4020](https://github.com/hashicorp/terraform/issues/4020))
  * provider/aws: Only set `weight` on an `aws_route53_record` if it has been set in configuration ([#3900](https://github.com/hashicorp/terraform/issues/3900))
  * provider/aws: Ignore association not existing on route table destroy ([#3615](https://github.com/hashicorp/terraform/issues/3615))
  * provider/aws: Fix policy encoding issue with SNS Topics ([#3700](https://github.com/hashicorp/terraform/issues/3700))
  * provider/aws: Correctly export ARN in `aws_iam_saml_provider` ([#3827](https://github.com/hashicorp/terraform/issues/3827))
  * provider/aws: Fix issue deleting users who are attached to a group ([#4005](https://github.com/hashicorp/terraform/issues/4005))
  * provider/aws: Fix crash in Route53 Record if Zone not found ([#3945](https://github.com/hashicorp/terraform/issues/3945))
  * provider/aws: Retry deleting IAM Server Cert on dependency violation ([#3898](https://github.com/hashicorp/terraform/issues/3898))
  * provider/aws: Update Spot Instance request to provide connection information ([#3940](https://github.com/hashicorp/terraform/issues/3940))
  * provider/aws: Fix typo in error checking for IAM Policy Attachments ([#3970](https://github.com/hashicorp/terraform/issues/3970))
  * provider/aws: Fix issue with LB Cookie Stickiness and empty expiration period ([#3908](https://github.com/hashicorp/terraform/issues/3908))
  * provider/aws: Tolerate ElastiCache clusters being deleted outside Terraform ([#3767](https://github.com/hashicorp/terraform/issues/3767))
  * provider/aws: Downcase Route 53 record names in state file to match API output ([#3574](https://github.com/hashicorp/terraform/issues/3574))
  * provider/aws: Fix issue that could occur if no ECS Cluster was found for a given name ([#3829](https://github.com/hashicorp/terraform/issues/3829))
  * provider/aws: Fix issue with SNS topic policy if omitted ([#3777](https://github.com/hashicorp/terraform/issues/3777))
  * provider/aws: Support scratch volumes in `aws_ecs_task_definition` ([#3810](https://github.com/hashicorp/terraform/issues/3810))
  * provider/aws: Treat `aws_ecs_service` w/ Status==INACTIVE as deleted ([#3828](https://github.com/hashicorp/terraform/issues/3828))
  * provider/aws: Expand ~ to homedir in `aws_s3_bucket_object.source` ([#3910](https://github.com/hashicorp/terraform/issues/3910))
  * provider/aws: Fix issue with updating the `aws_ecs_task_definition` where `aws_ecs_service` didn't wait for a new computed ARN ([#3924](https://github.com/hashicorp/terraform/issues/3924))
  * provider/aws: Prevent crashing when deleting `aws_ecs_service` that is already gone ([#3914](https://github.com/hashicorp/terraform/issues/3914))
  * provider/aws: Allow spaces in `aws_db_subnet_group.name` (undocumented in the API) ([#3955](https://github.com/hashicorp/terraform/issues/3955))
  * provider/aws: Make VPC ID required on subnets ([#4021](https://github.com/hashicorp/terraform/issues/4021))
  * provider/azure: Various bug fixes ([#3695](https://github.com/hashicorp/terraform/issues/3695))
  * provider/digitalocean: Fix issue preventing SSH fingerprints from working ([#3633](https://github.com/hashicorp/terraform/issues/3633))
  * provider/digitalocean: Fix the DigitalOcean Droplet 404 potential on refresh of state ([#3768](https://github.com/hashicorp/terraform/issues/3768))
  * provider/openstack: Fix several issues causing unresolvable diffs ([#3440](https://github.com/hashicorp/terraform/issues/3440))
  * provider/openstack: Safely delete security groups ([#3696](https://github.com/hashicorp/terraform/issues/3696))
  * provider/openstack: Ignore order of `security_groups` in instance ([#3651](https://github.com/hashicorp/terraform/issues/3651))
  * provider/vsphere: Fix d.SetConnInfo error in case of a missing IP address ([#3636](https://github.com/hashicorp/terraform/issues/3636))
  * provider/openstack: Fix boot from volume ([#3206](https://github.com/hashicorp/terraform/issues/3206))
  * provider/openstack: Fix crashing when image is no longer accessible ([#2189](https://github.com/hashicorp/terraform/issues/2189))
  * provider/openstack: Better handling of network resource state changes ([#3712](https://github.com/hashicorp/terraform/issues/3712))
  * provider/openstack: Fix crashing when no security group is specified ([#3801](https://github.com/hashicorp/terraform/issues/3801))
  * provider/packet: Fix issue that could cause errors when provisioning many devices at once ([#3847](https://github.com/hashicorp/terraform/issues/3847))
  * provider/packet: Fix connection information for devices, allowing provisioners to run ([#3948](https://github.com/hashicorp/terraform/issues/3948))
  * provider/openstack: Fix issue preventing security group rules from being removed ([#3796](https://github.com/hashicorp/terraform/issues/3796))
  * provider/template: `template_file`: source contents instead of path ([#3909](https://github.com/hashicorp/terraform/issues/3909))

## 0.6.6 (October 23, 2015)

FEATURES:

  * New interpolation functions: `cidrhost`, `cidrnetmask` and `cidrsubnet` ([#3127](https://github.com/hashicorp/terraform/issues/3127))

IMPROVEMENTS:

  * "forces new resource" now highlighted in plan output ([#3136](https://github.com/hashicorp/terraform/issues/3136))

BUG FIXES:

  * helper/schema: Better error message for assigning list/map to string ([#3009](https://github.com/hashicorp/terraform/issues/3009))
  * remote/state/atlas: Additional remote state conflict handling for semantically neutral state changes ([#3603](https://github.com/hashicorp/terraform/issues/3603))

## 0.6.5 (October 21, 2015)

FEATURES:

  * **New resources: `aws_codeploy_app` and `aws_codeploy_deployment_group`** ([#2783](https://github.com/hashicorp/terraform/issues/2783))
  * New remote state backend: `etcd` ([#3487](https://github.com/hashicorp/terraform/issues/3487))
  * New interpolation functions: `upper` and `lower` ([#3558](https://github.com/hashicorp/terraform/issues/3558))

BUG FIXES:

  * core: Fix remote state conflicts caused by ambiguity in ordering of deeply nested modules ([#3573](https://github.com/hashicorp/terraform/issues/3573))
  * core: Fix remote state conflicts caused by state metadata differences ([#3569](https://github.com/hashicorp/terraform/issues/3569))
  * core: Avoid using http.DefaultClient ([#3532](https://github.com/hashicorp/terraform/issues/3532))

INTERNAL IMPROVEMENTS:

  * provider/digitalocean: use official Go client ([#3333](https://github.com/hashicorp/terraform/issues/3333))
  * core: extract module fetching to external library ([#3516](https://github.com/hashicorp/terraform/issues/3516))

## 0.6.4 (October 15, 2015)

FEATURES:

  * **New provider: `rundeck`** ([#2412](https://github.com/hashicorp/terraform/issues/2412))
  * **New provider: `packet`** ([#2260](https://github.com/hashicorp/terraform/issues/2260)), ([#3472](https://github.com/hashicorp/terraform/issues/3472))
  * **New provider: `vsphere`**: Initial support for a VM resource ([#3419](https://github.com/hashicorp/terraform/issues/3419))
  * **New resource: `cloudstack_loadbalancer_rule`** ([#2934](https://github.com/hashicorp/terraform/issues/2934))
  * **New resource: `google_compute_project_metadata`** ([#3065](https://github.com/hashicorp/terraform/issues/3065))
  * **New resources: `aws_ami`, `aws_ami_copy`, `aws_ami_from_instance`** ([#2784](https://github.com/hashicorp/terraform/issues/2784))
  * **New resources: `aws_cloudwatch_log_group`** ([#2415](https://github.com/hashicorp/terraform/issues/2415))
  * **New resource: `google_storage_bucket_object`** ([#3192](https://github.com/hashicorp/terraform/issues/3192))
  * **New resources: `google_compute_vpn_gateway`, `google_compute_vpn_tunnel`** ([#3213](https://github.com/hashicorp/terraform/issues/3213))
  * **New resources: `google_storage_bucket_acl`, `google_storage_object_acl`** ([#3272](https://github.com/hashicorp/terraform/issues/3272))
  * **New resource: `aws_iam_saml_provider`** ([#3156](https://github.com/hashicorp/terraform/issues/3156))
  * **New resources: `aws_efs_file_system` and `aws_efs_mount_target`** ([#2196](https://github.com/hashicorp/terraform/issues/2196))
  * **New resources: `aws_opsworks_*`** ([#2162](https://github.com/hashicorp/terraform/issues/2162))
  * **New resource: `aws_elasticsearch_domain`** ([#3443](https://github.com/hashicorp/terraform/issues/3443))
  * **New resource: `aws_directory_service_directory`** ([#3228](https://github.com/hashicorp/terraform/issues/3228))
  * **New resource: `aws_autoscaling_lifecycle_hook`** ([#3351](https://github.com/hashicorp/terraform/issues/3351))
  * **New resource: `aws_placement_group`** ([#3457](https://github.com/hashicorp/terraform/issues/3457))
  * **New resource: `aws_glacier_vault`** ([#3491](https://github.com/hashicorp/terraform/issues/3491))
  * **New lifecycle flag: `ignore_changes`** ([#2525](https://github.com/hashicorp/terraform/issues/2525))

IMPROVEMENTS:

  * core: Add a function to find the index of an element in a list. ([#2704](https://github.com/hashicorp/terraform/issues/2704))
  * core: Print all outputs when `terraform output` is called with no arguments ([#2920](https://github.com/hashicorp/terraform/issues/2920))
  * core: In plan output summary, count resource replacement as Add/Remove instead of Change ([#3173](https://github.com/hashicorp/terraform/issues/3173))
  * core: Add interpolation functions for base64 encoding and decoding. ([#3325](https://github.com/hashicorp/terraform/issues/3325))
  * core: Expose parallelism as a CLI option instead of a hard-coding the default of 10 ([#3365](https://github.com/hashicorp/terraform/issues/3365))
  * core: Add interpolation function `compact`, to remove empty elements from a list. ([#3239](https://github.com/hashicorp/terraform/issues/3239)), ([#3479](https://github.com/hashicorp/terraform/issues/3479))
  * core: Allow filtering of log output by level, using e.g. ``TF_LOG=INFO`` ([#3380](https://github.com/hashicorp/terraform/issues/3380))
  * provider/aws: Add `instance_initiated_shutdown_behavior` to AWS Instance ([#2887](https://github.com/hashicorp/terraform/issues/2887))
  * provider/aws: Support IAM role names (previously just ARNs) in `aws_ecs_service.iam_role` ([#3061](https://github.com/hashicorp/terraform/issues/3061))
  * provider/aws: Add update method to RDS Subnet groups, can modify subnets without recreating  ([#3053](https://github.com/hashicorp/terraform/issues/3053))
  * provider/aws: Paginate notifications returned for ASG Notifications ([#3043](https://github.com/hashicorp/terraform/issues/3043))
  * provider/aws: Adds additional S3 Bucket Object inputs ([#3265](https://github.com/hashicorp/terraform/issues/3265))
  * provider/aws: add `ses_smtp_password` to `aws_iam_access_key` ([#3165](https://github.com/hashicorp/terraform/issues/3165))
  * provider/aws: read `iam_instance_profile` for `aws_instance` and save to state ([#3167](https://github.com/hashicorp/terraform/issues/3167))
  * provider/aws: allow `instance` to be computed in `aws_eip` ([#3036](https://github.com/hashicorp/terraform/issues/3036))
  * provider/aws: Add `versioning` option to `aws_s3_bucket` ([#2942](https://github.com/hashicorp/terraform/issues/2942))
  * provider/aws: Add `configuration_endpoint` to `aws_elasticache_cluster` ([#3250](https://github.com/hashicorp/terraform/issues/3250))
  * provider/aws: Add validation for `app_cookie_stickiness_policy.name` ([#3277](https://github.com/hashicorp/terraform/issues/3277))
  * provider/aws: Add validation for `db_parameter_group.name` ([#3279](https://github.com/hashicorp/terraform/issues/3279))
  * provider/aws: Set DynamoDB Table ARN after creation ([#3500](https://github.com/hashicorp/terraform/issues/3500))
  * provider/aws: `aws_s3_bucket_object` allows interpolated content to be set with new `content` attribute. ([#3200](https://github.com/hashicorp/terraform/issues/3200))
  * provider/aws: Allow tags for `aws_kinesis_stream` resource. ([#3397](https://github.com/hashicorp/terraform/issues/3397))
  * provider/aws: Configurable capacity waiting duration for ASGs ([#3191](https://github.com/hashicorp/terraform/issues/3191))
  * provider/aws: Allow non-persistent Spot Requests ([#3311](https://github.com/hashicorp/terraform/issues/3311))
  * provider/aws: Support tags for AWS DB subnet group ([#3138](https://github.com/hashicorp/terraform/issues/3138))
  * provider/cloudstack: Add `project` parameter to `cloudstack_vpc`, `cloudstack_network`, `cloudstack_ipaddress` and `cloudstack_disk` ([#3035](https://github.com/hashicorp/terraform/issues/3035))
  * provider/openstack: add functionality to attach FloatingIP to Port ([#1788](https://github.com/hashicorp/terraform/issues/1788))
  * provider/google: Can now do multi-region deployments without using multiple providers ([#3258](https://github.com/hashicorp/terraform/issues/3258))
  * remote/s3: Allow canned ACLs to be set on state objects. ([#3233](https://github.com/hashicorp/terraform/issues/3233))
  * remote/s3: Remote state is stored in S3 with `Content-Type: application/json` ([#3385](https://github.com/hashicorp/terraform/issues/3385))

BUG FIXES:

  * core: Fix problems referencing list attributes in interpolations ([#2157](https://github.com/hashicorp/terraform/issues/2157))
  * core: don't error on computed value during input walk ([#2988](https://github.com/hashicorp/terraform/issues/2988))
  * core: Ignore missing variables during destroy phase ([#3393](https://github.com/hashicorp/terraform/issues/3393))
  * provider/google: Crashes with interface conversion in GCE Instance Template ([#3027](https://github.com/hashicorp/terraform/issues/3027))
  * provider/google: Convert int to int64 when building the GKE cluster.NodeConfig struct ([#2978](https://github.com/hashicorp/terraform/issues/2978))
  * provider/google: google_compute_instance_template.network_interface.network should be a URL ([#3226](https://github.com/hashicorp/terraform/issues/3226))
  * provider/aws: Retry creation of `aws_ecs_service` if IAM policy isn't ready yet ([#3061](https://github.com/hashicorp/terraform/issues/3061))
  * provider/aws: Fix issue with mixed capitalization for RDS Instances  ([#3053](https://github.com/hashicorp/terraform/issues/3053))
  * provider/aws: Fix issue with RDS to allow major version upgrades ([#3053](https://github.com/hashicorp/terraform/issues/3053))
  * provider/aws: Fix shard_count in `aws_kinesis_stream` ([#2986](https://github.com/hashicorp/terraform/issues/2986))
  * provider/aws: Fix issue with `key_name` and using VPCs with spot instance requests ([#2954](https://github.com/hashicorp/terraform/issues/2954))
  * provider/aws: Fix unresolvable diffs coming from `aws_elasticache_cluster` names being downcased
      by AWS ([#3120](https://github.com/hashicorp/terraform/issues/3120))
  * provider/aws: Read instance source_dest_check and save to state ([#3152](https://github.com/hashicorp/terraform/issues/3152))
  * provider/aws: Allow `weight = 0` in Route53 records ([#3196](https://github.com/hashicorp/terraform/issues/3196))
  * provider/aws: Normalize aws_elasticache_cluster id to lowercase, allowing convergence. ([#3235](https://github.com/hashicorp/terraform/issues/3235))
  * provider/aws: Fix ValidateAccountId for IAM Instance Profiles ([#3313](https://github.com/hashicorp/terraform/issues/3313))
  * provider/aws: Update Security Group Rules to Version 2 ([#3019](https://github.com/hashicorp/terraform/issues/3019))
  * provider/aws: Migrate KeyPair to version 1, fixing issue with using `file()` ([#3470](https://github.com/hashicorp/terraform/issues/3470))
  * provider/aws: Fix force_delete on autoscaling groups ([#3485](https://github.com/hashicorp/terraform/issues/3485))
  * provider/aws: Fix crash with VPC Peering connections ([#3490](https://github.com/hashicorp/terraform/issues/3490))
  * provider/aws: fix bug with reading GSIs from dynamodb ([#3300](https://github.com/hashicorp/terraform/issues/3300))
  * provider/docker: Fix issue preventing private images from being referenced ([#2619](https://github.com/hashicorp/terraform/issues/2619))
  * provider/digitalocean: Fix issue causing unnecessary diffs based on droplet slugsize case ([#3284](https://github.com/hashicorp/terraform/issues/3284))
  * provider/openstack: add state 'downloading' to list of expected states in
      `blockstorage_volume_v1` creation ([#2866](https://github.com/hashicorp/terraform/issues/2866))
  * provider/openstack: remove security groups (by name) before adding security
      groups (by id) ([#2008](https://github.com/hashicorp/terraform/issues/2008))

INTERNAL IMPROVEMENTS:

  * core: Makefile target "plugin-dev" for building just one plugin. ([#3229](https://github.com/hashicorp/terraform/issues/3229))
  * helper/schema: Don't allow ``Update`` func if no attributes can actually be updated, per schema. ([#3288](https://github.com/hashicorp/terraform/issues/3288))
  * helper/schema: Default hashing function for sets ([#3018](https://github.com/hashicorp/terraform/issues/3018))
  * helper/multierror: Remove in favor of [github.com/hashicorp/go-multierror](http://github.com/hashicorp/go-multierror). ([#3336](https://github.com/hashicorp/terraform/issues/3336))

## 0.6.3 (August 11, 2015)

BUG FIXES:

  * core: Skip all descendents after error, not just children; helps prevent confusing
      additional errors/crashes after initial failure ([#2963](https://github.com/hashicorp/terraform/issues/2963))
  * core: fix deadlock possibility when both a module and a dependent resource are
      removed in the same run ([#2968](https://github.com/hashicorp/terraform/issues/2968))
  * provider/aws: Fix issue with authenticating when using IAM profiles ([#2959](https://github.com/hashicorp/terraform/issues/2959))

## 0.6.2 (August 6, 2015)

FEATURES:

  * **New resource: `google_compute_instance_group_manager`** ([#2868](https://github.com/hashicorp/terraform/issues/2868))
  * **New resource: `google_compute_autoscaler`** ([#2868](https://github.com/hashicorp/terraform/issues/2868))
  * **New resource: `aws_s3_bucket_object`** ([#2898](https://github.com/hashicorp/terraform/issues/2898))

IMPROVEMENTS:

  * core: Add resource IDs to errors coming from `apply`/`refresh` ([#2815](https://github.com/hashicorp/terraform/issues/2815))
  * provider/aws: Validate credentials before walking the graph ([#2730](https://github.com/hashicorp/terraform/issues/2730))
  * provider/aws: Added website_domain for S3 buckets ([#2210](https://github.com/hashicorp/terraform/issues/2210))
  * provider/aws: ELB names are now optional, and generated by Terraform if omitted ([#2571](https://github.com/hashicorp/terraform/issues/2571))
  * provider/aws: Downcase RDS engine names to prevent continuous diffs ([#2745](https://github.com/hashicorp/terraform/issues/2745))
  * provider/aws: Added `source_dest_check` attribute to the aws_network_interface ([#2741](https://github.com/hashicorp/terraform/issues/2741))
  * provider/aws: Clean up externally removed Launch Configurations ([#2806](https://github.com/hashicorp/terraform/issues/2806))
  * provider/aws: Allow configuration of the DynamoDB Endpoint ([#2825](https://github.com/hashicorp/terraform/issues/2825))
  * provider/aws: Compute private ip addresses of ENIs if they are not specified ([#2743](https://github.com/hashicorp/terraform/issues/2743))
  * provider/aws: Add `arn` attribute for DynamoDB tables ([#2924](https://github.com/hashicorp/terraform/issues/2924))
  * provider/aws: Fail silently when account validation fails while from instance profile ([#3001](https://github.com/hashicorp/terraform/issues/3001))
  * provider/azure: Allow `settings_file` to accept XML string ([#2922](https://github.com/hashicorp/terraform/issues/2922))
  * provider/azure: Provide a simpler error when using a Platform Image without a
      Storage Service ([#2861](https://github.com/hashicorp/terraform/issues/2861))
  * provider/google: `account_file` is now expected to be JSON. Paths are still supported for
      backwards compatibility. ([#2839](https://github.com/hashicorp/terraform/issues/2839))

BUG FIXES:

  * core: Prevent error duplication in `apply` ([#2815](https://github.com/hashicorp/terraform/issues/2815))
  * core: Fix crash when  a provider validation adds a warning ([#2878](https://github.com/hashicorp/terraform/issues/2878))
  * provider/aws: Fix issue with toggling monitoring in AWS Instances ([#2794](https://github.com/hashicorp/terraform/issues/2794))
  * provider/aws: Fix issue with Spot Instance Requests and cancellation ([#2805](https://github.com/hashicorp/terraform/issues/2805))
  * provider/aws: Fix issue with checking for ElastiCache cluster cache node status ([#2842](https://github.com/hashicorp/terraform/issues/2842))
  * provider/aws: Fix issue when unable to find a Root Block Device name of an Instance Backed
      AMI ([#2646](https://github.com/hashicorp/terraform/issues/2646))
  * provider/dnsimple: Domain and type should force new records ([#2777](https://github.com/hashicorp/terraform/issues/2777))
  * provider/aws: Fix issue with IAM Server Certificates and Chains ([#2871](https://github.com/hashicorp/terraform/issues/2871))
  * provider/aws: Fix issue with IAM Server Certificates when using `path` ([#2871](https://github.com/hashicorp/terraform/issues/2871))
  * provider/aws: Fix issue in Security Group Rules when the Security Group is not found ([#2897](https://github.com/hashicorp/terraform/issues/2897))
  * provider/aws: allow external ENI attachments ([#2943](https://github.com/hashicorp/terraform/issues/2943))
  * provider/aws: Fix issue with S3 Buckets, and throwing an error when not found ([#2925](https://github.com/hashicorp/terraform/issues/2925))

## 0.6.1 (July 20, 2015)

FEATURES:

  * **New resource: `google_container_cluster`** ([#2357](https://github.com/hashicorp/terraform/issues/2357))
  * **New resource: `aws_vpc_endpoint`** ([#2695](https://github.com/hashicorp/terraform/issues/2695))

IMPROVEMENTS:

  * connection/ssh: Print SSH bastion host details to output ([#2684](https://github.com/hashicorp/terraform/issues/2684))
  * provider/aws: Create RDS databases from snapshots ([#2062](https://github.com/hashicorp/terraform/issues/2062))
  * provider/aws: Add support for restoring from Redis backup stored in S3 ([#2634](https://github.com/hashicorp/terraform/issues/2634))
  * provider/aws: Add `maintenance_window` to ElastiCache cluster ([#2642](https://github.com/hashicorp/terraform/issues/2642))
  * provider/aws: Availability Zones are optional when specifying VPC Zone Identifiers in
      Auto Scaling Groups updates ([#2724](https://github.com/hashicorp/terraform/issues/2724))
  * provider/google: Add metadata_startup_script to google_compute_instance ([#2375](https://github.com/hashicorp/terraform/issues/2375))

BUG FIXES:

  * core: Don't prompt for variables with defaults ([#2613](https://github.com/hashicorp/terraform/issues/2613))
  * core: Return correct number of planned updates ([#2620](https://github.com/hashicorp/terraform/issues/2620))
  * core: Fix "provider not found" error that can occur while running
      a destroy plan with grandchildren modules ([#2755](https://github.com/hashicorp/terraform/issues/2755))
  * core: Fix UUID showing up in diff for computed splat (`foo.*.bar`)
      variables. ([#2788](https://github.com/hashicorp/terraform/issues/2788))
  * core: Orphan modules that contain no resources (only other modules)
      are properly destroyed up to arbitrary depth ([#2786](https://github.com/hashicorp/terraform/issues/2786))
  * core: Fix "attribute not available" during destroy plans in
      cases where the parameter is passed between modules ([#2775](https://github.com/hashicorp/terraform/issues/2775))
  * core: Record schema version when destroy fails ([#2923](https://github.com/hashicorp/terraform/issues/2923))
  * connection/ssh: fix issue on machines with an SSH Agent available
    preventing `key_file` from being read without explicitly
    setting `agent = false` ([#2615](https://github.com/hashicorp/terraform/issues/2615))
  * provider/aws: Allow uppercase characters in `aws_elb.name` ([#2580](https://github.com/hashicorp/terraform/issues/2580))
  * provider/aws: Allow underscores in `aws_db_subnet_group.name` (undocumented by AWS) ([#2604](https://github.com/hashicorp/terraform/issues/2604))
  * provider/aws: Allow dots in `aws_db_subnet_group.name` (undocumented by AWS) ([#2665](https://github.com/hashicorp/terraform/issues/2665))
  * provider/aws: Fix issue with pending Spot Instance requests ([#2640](https://github.com/hashicorp/terraform/issues/2640))
  * provider/aws: Fix issue in AWS Classic environment with referencing external
      Security Groups ([#2644](https://github.com/hashicorp/terraform/issues/2644))
  * provider/aws: Bump internet gateway detach timeout ([#2669](https://github.com/hashicorp/terraform/issues/2669))
  * provider/aws: Fix issue with detecting differences in DB Parameters ([#2728](https://github.com/hashicorp/terraform/issues/2728))
  * provider/aws: `ecs_cluster` rename (recreation) and deletion is handled correctly ([#2698](https://github.com/hashicorp/terraform/issues/2698))
  * provider/aws: `aws_route_table` ignores routes generated for VPC endpoints ([#2695](https://github.com/hashicorp/terraform/issues/2695))
  * provider/aws: Fix issue with Launch Configurations and enable_monitoring ([#2735](https://github.com/hashicorp/terraform/issues/2735))
  * provider/openstack: allow empty api_key and endpoint_type ([#2626](https://github.com/hashicorp/terraform/issues/2626))
  * provisioner/chef: Fix permission denied error with ohai hints ([#2781](https://github.com/hashicorp/terraform/issues/2781))

## 0.6.0 (June 30, 2015)

BACKWARDS INCOMPATIBILITIES:

 * command/push: If a variable is already set within Atlas, it won't be
     updated unless the `-overwrite` flag is present ([#2373](https://github.com/hashicorp/terraform/issues/2373))
 * connection/ssh: The `agent` field now defaults to `true` if
     the `SSH_AGENT_SOCK` environment variable is present. In other words,
     `ssh-agent` support is now opt-out instead of opt-in functionality. ([#2408](https://github.com/hashicorp/terraform/issues/2408))
 * provider/aws: If you were setting access and secret key to blank ("")
     to force Terraform to load credentials from another source such as the
     EC2 role, this will now error. Remove the blank lines and Terraform
     will load from other sources.
 * `concat()` has been repurposed to combine lists instead of strings (old behavior
     of joining strings is maintained in this version but is deprecated, strings
     should be combined using interpolation syntax, like "${var.foo}{var.bar}")
     ([#1790](https://github.com/hashicorp/terraform/issues/1790))

FEATURES:

  * **New provider: `azure`** ([#2052](https://github.com/hashicorp/terraform/issues/2052), [#2053](https://github.com/hashicorp/terraform/issues/2053), [#2372](https://github.com/hashicorp/terraform/issues/2372), [#2380](https://github.com/hashicorp/terraform/issues/2380), [#2394](https://github.com/hashicorp/terraform/issues/2394), [#2515](https://github.com/hashicorp/terraform/issues/2515), [#2530](https://github.com/hashicorp/terraform/issues/2530), [#2562](https://github.com/hashicorp/terraform/issues/2562))
  * **New resource: `aws_autoscaling_notification`** ([#2197](https://github.com/hashicorp/terraform/issues/2197))
  * **New resource: `aws_autoscaling_policy`** ([#2201](https://github.com/hashicorp/terraform/issues/2201))
  * **New resource: `aws_cloudwatch_metric_alarm`** ([#2201](https://github.com/hashicorp/terraform/issues/2201))
  * **New resource: `aws_dynamodb_table`** ([#2121](https://github.com/hashicorp/terraform/issues/2121))
  * **New resource: `aws_ecs_cluster`** ([#1803](https://github.com/hashicorp/terraform/issues/1803))
  * **New resource: `aws_ecs_service`** ([#1803](https://github.com/hashicorp/terraform/issues/1803))
  * **New resource: `aws_ecs_task_definition`** ([#1803](https://github.com/hashicorp/terraform/issues/1803), [#2402](https://github.com/hashicorp/terraform/issues/2402))
  * **New resource: `aws_elasticache_parameter_group`** ([#2276](https://github.com/hashicorp/terraform/issues/2276))
  * **New resource: `aws_flow_log`** ([#2384](https://github.com/hashicorp/terraform/issues/2384))
  * **New resource: `aws_iam_group_association`** ([#2273](https://github.com/hashicorp/terraform/issues/2273))
  * **New resource: `aws_iam_policy_attachment`** ([#2395](https://github.com/hashicorp/terraform/issues/2395))
  * **New resource: `aws_lambda_function`** ([#2170](https://github.com/hashicorp/terraform/issues/2170))
  * **New resource: `aws_route53_delegation_set`** ([#1999](https://github.com/hashicorp/terraform/issues/1999))
  * **New resource: `aws_route53_health_check`** ([#2226](https://github.com/hashicorp/terraform/issues/2226))
  * **New resource: `aws_spot_instance_request`** ([#2263](https://github.com/hashicorp/terraform/issues/2263))
  * **New resource: `cloudstack_ssh_keypair`** ([#2004](https://github.com/hashicorp/terraform/issues/2004))
  * **New remote state backend: `swift`**: You can now store remote state in
     a OpenStack Swift. ([#2254](https://github.com/hashicorp/terraform/issues/2254))
  * command/output: support display of module outputs ([#2102](https://github.com/hashicorp/terraform/issues/2102))
  * core: `keys()` and `values()` funcs for map variables ([#2198](https://github.com/hashicorp/terraform/issues/2198))
  * connection/ssh: SSH bastion host support and ssh-agent forwarding ([#2425](https://github.com/hashicorp/terraform/issues/2425))

IMPROVEMENTS:

  * core: HTTP remote state now accepts `skip_cert_verification`
      option to ignore TLS cert verification. ([#2214](https://github.com/hashicorp/terraform/issues/2214))
  * core: S3 remote state now accepts the 'encrypt' option for SSE ([#2405](https://github.com/hashicorp/terraform/issues/2405))
  * core: `plan` now reports sum of resources to be changed/created/destroyed ([#2458](https://github.com/hashicorp/terraform/issues/2458))
  * core: Change string list representation so we can distinguish empty, single
      element lists ([#2504](https://github.com/hashicorp/terraform/issues/2504))
  * core: Properly close provider and provisioner plugin connections ([#2406](https://github.com/hashicorp/terraform/issues/2406), [#2527](https://github.com/hashicorp/terraform/issues/2527))
  * provider/aws: AutoScaling groups now support updating Load Balancers without
      recreation ([#2472](https://github.com/hashicorp/terraform/issues/2472))
  * provider/aws: Allow more in-place updates for ElastiCache cluster without recreating
      ([#2469](https://github.com/hashicorp/terraform/issues/2469))
  * provider/aws: ElastiCache Subnet Groups can be updated
      without destroying first ([#2191](https://github.com/hashicorp/terraform/issues/2191))
  * provider/aws: Normalize `certificate_chain` in `aws_iam_server_certificate` to
      prevent unnecessary replacement. ([#2411](https://github.com/hashicorp/terraform/issues/2411))
  * provider/aws: `aws_instance` supports `monitoring' ([#2489](https://github.com/hashicorp/terraform/issues/2489))
  * provider/aws: `aws_launch_configuration` now supports `enable_monitoring` ([#2410](https://github.com/hashicorp/terraform/issues/2410))
  * provider/aws: Show outputs after `terraform refresh` ([#2347](https://github.com/hashicorp/terraform/issues/2347))
  * provider/aws: Add backoff/throttling during DynamoDB creation ([#2462](https://github.com/hashicorp/terraform/issues/2462))
  * provider/aws: Add validation for aws_vpc.cidr_block ([#2514](https://github.com/hashicorp/terraform/issues/2514))
  * provider/aws: Add validation for aws_db_subnet_group.name ([#2513](https://github.com/hashicorp/terraform/issues/2513))
  * provider/aws: Add validation for aws_db_instance.identifier ([#2516](https://github.com/hashicorp/terraform/issues/2516))
  * provider/aws: Add validation for aws_elb.name ([#2517](https://github.com/hashicorp/terraform/issues/2517))
  * provider/aws: Add validation for aws_security_group (name+description) ([#2518](https://github.com/hashicorp/terraform/issues/2518))
  * provider/aws: Add validation for aws_launch_configuration ([#2519](https://github.com/hashicorp/terraform/issues/2519))
  * provider/aws: Add validation for aws_autoscaling_group.name ([#2520](https://github.com/hashicorp/terraform/issues/2520))
  * provider/aws: Add validation for aws_iam_role.name ([#2521](https://github.com/hashicorp/terraform/issues/2521))
  * provider/aws: Add validation for aws_iam_role_policy.name ([#2552](https://github.com/hashicorp/terraform/issues/2552))
  * provider/aws: Add validation for aws_iam_instance_profile.name ([#2553](https://github.com/hashicorp/terraform/issues/2553))
  * provider/aws: aws_auto_scaling_group.default_cooldown no longer requires
      resource replacement ([#2510](https://github.com/hashicorp/terraform/issues/2510))
  * provider/aws: add AH and ESP protocol integers ([#2321](https://github.com/hashicorp/terraform/issues/2321))
  * provider/docker: `docker_container` has the `privileged`
      option. ([#2227](https://github.com/hashicorp/terraform/issues/2227))
  * provider/openstack: allow `OS_AUTH_TOKEN` environment variable
      to set the openstack `api_key` field ([#2234](https://github.com/hashicorp/terraform/issues/2234))
  * provider/openstack: Can now configure endpoint type (public, admin,
      internal) ([#2262](https://github.com/hashicorp/terraform/issues/2262))
  * provider/cloudstack: `cloudstack_instance` now supports projects ([#2115](https://github.com/hashicorp/terraform/issues/2115))
  * provisioner/chef: Added a `os_type` to specifically specify the target OS ([#2483](https://github.com/hashicorp/terraform/issues/2483))
  * provisioner/chef: Added a `ohai_hints` option to upload hint files ([#2487](https://github.com/hashicorp/terraform/issues/2487))

BUG FIXES:

  * core: lifecycle `prevent_destroy` can be any value that can be
      coerced into a bool ([#2268](https://github.com/hashicorp/terraform/issues/2268))
  * core: matching provider types in sibling modules won't override
      each other's config. ([#2464](https://github.com/hashicorp/terraform/issues/2464))
  * core: computed provider configurations now properly validate ([#2457](https://github.com/hashicorp/terraform/issues/2457))
  * core: orphan (commented out) resource dependencies are destroyed in
      the correct order ([#2453](https://github.com/hashicorp/terraform/issues/2453))
  * core: validate object types in plugins are actually objects ([#2450](https://github.com/hashicorp/terraform/issues/2450))
  * core: fix `-no-color` flag in subcommands ([#2414](https://github.com/hashicorp/terraform/issues/2414))
  * core: Fix error of 'attribute not found for variable' when a computed
      resource attribute is used as a parameter to a module ([#2477](https://github.com/hashicorp/terraform/issues/2477))
  * core: moduled orphans will properly inherit provider configs ([#2476](https://github.com/hashicorp/terraform/issues/2476))
  * core: modules with provider aliases work properly if the parent
      doesn't implement those aliases ([#2475](https://github.com/hashicorp/terraform/issues/2475))
  * core: unknown resource attributes passed in as parameters to modules
      now error ([#2478](https://github.com/hashicorp/terraform/issues/2478))
  * core: better error messages for missing variables ([#2479](https://github.com/hashicorp/terraform/issues/2479))
  * core: removed set items now properly appear in diffs and applies ([#2507](https://github.com/hashicorp/terraform/issues/2507))
  * core: '*' will not be added as part of the variable name when you
      attempt multiplication without a space ([#2505](https://github.com/hashicorp/terraform/issues/2505))
  * core: fix target dependency calculation across module boundaries ([#2555](https://github.com/hashicorp/terraform/issues/2555))
  * command/*: fixed bug where variable input was not asked for unset
      vars if terraform.tfvars existed ([#2502](https://github.com/hashicorp/terraform/issues/2502))
  * command/apply: prevent output duplication when reporting errors ([#2267](https://github.com/hashicorp/terraform/issues/2267))
  * command/apply: destroyed orphan resources are properly counted ([#2506](https://github.com/hashicorp/terraform/issues/2506))
  * provider/aws: loading credentials from the environment (vars, EC2 role,
      etc.) is more robust and will not ask for credentials from stdin ([#1841](https://github.com/hashicorp/terraform/issues/1841))
  * provider/aws: fix panic when route has no `cidr_block` ([#2215](https://github.com/hashicorp/terraform/issues/2215))
  * provider/aws: fix issue preventing destruction of IAM Roles ([#2177](https://github.com/hashicorp/terraform/issues/2177))
  * provider/aws: fix issue where Security Group Rules could collide and fail
      to save to the state file correctly ([#2376](https://github.com/hashicorp/terraform/issues/2376))
  * provider/aws: fix issue preventing destruction self referencing Securtity
     Group Rules ([#2305](https://github.com/hashicorp/terraform/issues/2305))
  * provider/aws: fix issue causing perpetual diff on ELB listeners
      when non-lowercase protocol strings were used ([#2246](https://github.com/hashicorp/terraform/issues/2246))
  * provider/aws: corrected frankfurt S3 website region ([#2259](https://github.com/hashicorp/terraform/issues/2259))
  * provider/aws: `aws_elasticache_cluster` port is required ([#2160](https://github.com/hashicorp/terraform/issues/2160))
  * provider/aws: Handle AMIs where RootBlockDevice does not appear in the
      BlockDeviceMapping, preventing root_block_device from working ([#2271](https://github.com/hashicorp/terraform/issues/2271))
  * provider/aws: fix `terraform show` with remote state ([#2371](https://github.com/hashicorp/terraform/issues/2371))
  * provider/aws: detect `instance_type` drift on `aws_instance` ([#2374](https://github.com/hashicorp/terraform/issues/2374))
  * provider/aws: fix crash when `security_group_rule` referenced non-existent
      security group ([#2434](https://github.com/hashicorp/terraform/issues/2434))
  * provider/aws: `aws_launch_configuration` retries if IAM instance
      profile is not ready yet. ([#2452](https://github.com/hashicorp/terraform/issues/2452))
  * provider/aws: `fqdn` is populated during creation for `aws_route53_record` ([#2528](https://github.com/hashicorp/terraform/issues/2528))
  * provider/aws: retry VPC delete on DependencyViolation due to eventual
      consistency ([#2532](https://github.com/hashicorp/terraform/issues/2532))
  * provider/aws: VPC peering connections in "failed" state are deleted ([#2544](https://github.com/hashicorp/terraform/issues/2544))
  * provider/aws: EIP deletion works if it was manually disassociated ([#2543](https://github.com/hashicorp/terraform/issues/2543))
  * provider/aws: `elasticache_subnet_group.subnet_ids` is now a required argument ([#2534](https://github.com/hashicorp/terraform/issues/2534))
  * provider/aws: handle nil response from VPN connection describes ([#2533](https://github.com/hashicorp/terraform/issues/2533))
  * provider/cloudflare: manual record deletion doesn't cause error ([#2545](https://github.com/hashicorp/terraform/issues/2545))
  * provider/digitalocean: handle case where droplet is deleted outside of
      terraform ([#2497](https://github.com/hashicorp/terraform/issues/2497))
  * provider/dme: No longer an error if record deleted manually ([#2546](https://github.com/hashicorp/terraform/issues/2546))
  * provider/docker: Fix issues when using containers with links ([#2327](https://github.com/hashicorp/terraform/issues/2327))
  * provider/openstack: fix panic case if API returns nil network ([#2448](https://github.com/hashicorp/terraform/issues/2448))
  * provider/template: fix issue causing "unknown variable" rendering errors
      when an existing set of template variables is changed ([#2386](https://github.com/hashicorp/terraform/issues/2386))
  * provisioner/chef: improve the decoding logic to prevent parameter not found errors ([#2206](https://github.com/hashicorp/terraform/issues/2206))

## 0.5.3 (June 1, 2015)

IMPROVEMENTS:

  * **New resource: `aws_kinesis_stream`** ([#2110](https://github.com/hashicorp/terraform/issues/2110))
  * **New resource: `aws_iam_server_certificate`** ([#2086](https://github.com/hashicorp/terraform/issues/2086))
  * **New resource: `aws_sqs_queue`** ([#1939](https://github.com/hashicorp/terraform/issues/1939))
  * **New resource: `aws_sns_topic`** ([#1974](https://github.com/hashicorp/terraform/issues/1974))
  * **New resource: `aws_sns_topic_subscription`** ([#1974](https://github.com/hashicorp/terraform/issues/1974))
  * **New resource: `aws_volume_attachment`** ([#2050](https://github.com/hashicorp/terraform/issues/2050))
  * **New resource: `google_storage_bucket`** ([#2060](https://github.com/hashicorp/terraform/issues/2060))
  * provider/aws: support ec2 termination protection ([#1988](https://github.com/hashicorp/terraform/issues/1988))
  * provider/aws: support for RDS Read Replicas ([#1946](https://github.com/hashicorp/terraform/issues/1946))
  * provider/aws: `aws_s3_bucket` add support for `policy` ([#1992](https://github.com/hashicorp/terraform/issues/1992))
  * provider/aws: `aws_ebs_volume` add support for `tags` ([#2135](https://github.com/hashicorp/terraform/issues/2135))
  * provider/aws: `aws_elasticache_cluster` Confirm node status before reporting
      available
  * provider/aws: `aws_network_acl` Add support for ICMP Protocol ([#2148](https://github.com/hashicorp/terraform/issues/2148))
  * provider/aws: New `force_destroy` parameter for S3 buckets, to destroy
      Buckets that contain objects ([#2007](https://github.com/hashicorp/terraform/issues/2007))
  * provider/aws: switching `health_check_type` on ASGs no longer requires
      resource refresh ([#2147](https://github.com/hashicorp/terraform/issues/2147))
  * provider/aws: ignore empty `vpc_security_group_ids` on `aws_instance` ([#2311](https://github.com/hashicorp/terraform/issues/2311))

BUG FIXES:

  * provider/aws: Correctly handle AWS keypairs which no longer exist ([#2032](https://github.com/hashicorp/terraform/issues/2032))
  * provider/aws: Fix issue with restoring an Instance from snapshot ID ([#2120](https://github.com/hashicorp/terraform/issues/2120))
  * provider/template: store relative path in the state ([#2038](https://github.com/hashicorp/terraform/issues/2038))
  * provisioner/chef: fix interpolation in the Chef provisioner ([#2168](https://github.com/hashicorp/terraform/issues/2168))
  * provisioner/remote-exec: Don't prepend shebang on scripts that already
      have one ([#2041](https://github.com/hashicorp/terraform/issues/2041))

## 0.5.2 (May 15, 2015)

FEATURES:

  * **Chef provisioning**: You can now provision new hosts (both Linux and
     Windows) with [Chef](https://chef.io) using a native provisioner ([#1868](https://github.com/hashicorp/terraform/issues/1868))

IMPROVEMENTS:

  * **New config function: `formatlist`** - Format lists in a similar way to `format`.
    Useful for creating URLs from a list of IPs. ([#1829](https://github.com/hashicorp/terraform/issues/1829))
  * **New resource: `aws_route53_zone_association`**
  * provider/aws: `aws_autoscaling_group` can wait for capacity in ELB
      via `min_elb_capacity` ([#1970](https://github.com/hashicorp/terraform/issues/1970))
  * provider/aws: `aws_db_instances` supports `license_model` ([#1966](https://github.com/hashicorp/terraform/issues/1966))
  * provider/aws: `aws_elasticache_cluster` add support for Tags ([#1965](https://github.com/hashicorp/terraform/issues/1965))
  * provider/aws: `aws_network_acl` Network ACLs can be applied to multiple subnets ([#1931](https://github.com/hashicorp/terraform/issues/1931))
  * provider/aws: `aws_s3_bucket` exports `hosted_zone_id` and `region` ([#1865](https://github.com/hashicorp/terraform/issues/1865))
  * provider/aws: `aws_s3_bucket` add support for website `redirect_all_requests_to` ([#1909](https://github.com/hashicorp/terraform/issues/1909))
  * provider/aws: `aws_route53_record` exports `fqdn` ([#1847](https://github.com/hashicorp/terraform/issues/1847))
  * provider/aws: `aws_route53_zone` can create private hosted zones ([#1526](https://github.com/hashicorp/terraform/issues/1526))
  * provider/google: `google_compute_instance` `scratch` attribute added ([#1920](https://github.com/hashicorp/terraform/issues/1920))

BUG FIXES:

  * core: fix "resource not found" for interpolation issues with modules
  * core: fix unflattenable error for orphans ([#1922](https://github.com/hashicorp/terraform/issues/1922))
  * core: fix deadlock with create-before-destroy + modules ([#1949](https://github.com/hashicorp/terraform/issues/1949))
  * core: fix "no roots found" error with create-before-destroy ([#1953](https://github.com/hashicorp/terraform/issues/1953))
  * core: variables set with environment variables won't validate as
      not set without a default ([#1930](https://github.com/hashicorp/terraform/issues/1930))
  * core: resources with a blank ID in the state are now assumed to not exist ([#1905](https://github.com/hashicorp/terraform/issues/1905))
  * command/push: local vars override remote ones ([#1881](https://github.com/hashicorp/terraform/issues/1881))
  * provider/aws: Mark `aws_security_group` description as `ForceNew` ([#1871](https://github.com/hashicorp/terraform/issues/1871))
  * provider/aws: `aws_db_instance` ARN value is correct ([#1910](https://github.com/hashicorp/terraform/issues/1910))
  * provider/aws: `aws_db_instance` only submit modify request if there
      is a change. ([#1906](https://github.com/hashicorp/terraform/issues/1906))
  * provider/aws: `aws_elasticache_cluster` export missing information on cluster nodes ([#1965](https://github.com/hashicorp/terraform/issues/1965))
  * provider/aws: bad AMI on a launch configuration won't block refresh ([#1901](https://github.com/hashicorp/terraform/issues/1901))
  * provider/aws: `aws_security_group` + `aws_subnet` - destroy timeout increased
    to prevent DependencyViolation errors. ([#1886](https://github.com/hashicorp/terraform/issues/1886))
  * provider/google: `google_compute_instance` Local SSDs no-longer cause crash
      ([#1088](https://github.com/hashicorp/terraform/issues/1088))
  * provider/google: `google_http_health_check` Defaults now driven from Terraform,
      avoids errors on update ([#1894](https://github.com/hashicorp/terraform/issues/1894))
  * provider/google: `google_compute_template` Update Instance Template network
      definition to match changes to Instance ([#980](https://github.com/hashicorp/terraform/issues/980))
  * provider/template: Fix infinite diff ([#1898](https://github.com/hashicorp/terraform/issues/1898))

## 0.5.1 (never released)

This version was never released since we accidentally skipped it!

## 0.5.0 (May 7, 2015)

BACKWARDS INCOMPATIBILITIES:

  * provider/aws: Terraform now remove the default egress rule created by AWS in
    a new security group.

FEATURES:

  * **Multi-provider (a.k.a multi-region)**: Multiple instances of a single
     provider can be configured so resources can apply to different settings.
     As an example, this allows Terraform to manage multiple regions with AWS.
  * **Environmental variables to set variables**: Environment variables can be
     used to set variables. The environment variables must be in the format
     `TF_VAR_name` and this will be checked last for a value.
  * **New remote state backend: `s3`**: You can now store remote state in
     an S3 bucket. ([#1723](https://github.com/hashicorp/terraform/issues/1723))
  * **Automatic AWS retries**: This release includes a lot of improvement
     around automatic retries of transient errors in AWS. The number of
     retry attempts is also configurable.
  * **Templates**: A new `template_file` resource allows long strings needing
     variable interpolation to be moved into files. ([#1778](https://github.com/hashicorp/terraform/issues/1778))
  * **Provision with WinRM**: Provisioners can now run remote commands on
     Windows hosts. ([#1483](https://github.com/hashicorp/terraform/issues/1483))

IMPROVEMENTS:

  * **New config function: `length`** - Get the length of a string or a list.
      Useful in conjunction with `split`. ([#1495](https://github.com/hashicorp/terraform/issues/1495))
  * **New resource: `aws_app_cookie_stickiness_policy`**
  * **New resource: `aws_customer_gateway`**
  * **New resource: `aws_ebs_volume`**
  * **New resource: `aws_elasticache_cluster`**
  * **New resource: `aws_elasticache_security_group`**
  * **New resource: `aws_elasticache_subnet_group`**
  * **New resource: `aws_iam_access_key`**
  * **New resource: `aws_iam_group_policy`**
  * **New resource: `aws_iam_group`**
  * **New resource: `aws_iam_instance_profile`**
  * **New resource: `aws_iam_policy`**
  * **New resource: `aws_iam_role_policy`**
  * **New resource: `aws_iam_role`**
  * **New resource: `aws_iam_user_policy`**
  * **New resource: `aws_iam_user`**
  * **New resource: `aws_lb_cookie_stickiness_policy`**
  * **New resource: `aws_proxy_protocol_policy`**
  * **New resource: `aws_security_group_rule`**
  * **New resource: `aws_vpc_dhcp_options_association`**
  * **New resource: `aws_vpc_dhcp_options`**
  * **New resource: `aws_vpn_connection_route`**
  * **New resource: `google_dns_managed_zone`**
  * **New resource: `google_dns_record_set`**
  * **Migrate to upstream AWS SDK:** Migrate the AWS provider to
      [awslabs/aws-sdk-go](https://github.com/awslabs/aws-sdk-go),
      the official `awslabs` library. Previously we had forked the library for
      stability while `awslabs` refactored. Now that work has completed, and we've
      migrated back to the upstream version.
  * core: Improve error message on diff mismatch ([#1501](https://github.com/hashicorp/terraform/issues/1501))
  * provisioner/file: expand `~` in source path ([#1569](https://github.com/hashicorp/terraform/issues/1569))
  * provider/aws: Better retry logic, now retries up to 11 times by default
      with exponentional backoff. This number is configurable. ([#1787](https://github.com/hashicorp/terraform/issues/1787))
  * provider/aws: Improved credential detection ([#1470](https://github.com/hashicorp/terraform/issues/1470))
  * provider/aws: Can specify a `token` via the config file ([#1601](https://github.com/hashicorp/terraform/issues/1601))
  * provider/aws: Added new `vpc_security_group_ids` attribute for AWS
      Instances. If using a VPC, you can now modify the security groups for that
      Instance without destroying it ([#1539](https://github.com/hashicorp/terraform/issues/1539))
  * provider/aws: White or blacklist account IDs that can be used to
      protect against accidents. ([#1595](https://github.com/hashicorp/terraform/issues/1595))
  * provider/aws: Add a subset of IAM resources ([#939](https://github.com/hashicorp/terraform/issues/939))
  * provider/aws: `aws_autoscaling_group` retries deletes through "in progress"
      errors ([#1840](https://github.com/hashicorp/terraform/issues/1840))
  * provider/aws: `aws_autoscaling_group` waits for healthy capacity during
      ASG creation ([#1839](https://github.com/hashicorp/terraform/issues/1839))
  * provider/aws: `aws_instance` supports placement groups ([#1358](https://github.com/hashicorp/terraform/issues/1358))
  * provider/aws: `aws_eip` supports network interface attachment ([#1681](https://github.com/hashicorp/terraform/issues/1681))
  * provider/aws: `aws_elb` supports in-place changing of listeners ([#1619](https://github.com/hashicorp/terraform/issues/1619))
  * provider/aws: `aws_elb` supports connection draining settings ([#1502](https://github.com/hashicorp/terraform/issues/1502))
  * provider/aws: `aws_elb` increase default idle timeout to 60s ([#1646](https://github.com/hashicorp/terraform/issues/1646))
  * provider/aws: `aws_key_pair` name can be omitted and generated ([#1751](https://github.com/hashicorp/terraform/issues/1751))
  * provider/aws: `aws_network_acl` improved validation for network ACL ports
      and protocols ([#1798](https://github.com/hashicorp/terraform/issues/1798)) ([#1808](https://github.com/hashicorp/terraform/issues/1808))
  * provider/aws: `aws_route_table` can target network interfaces ([#968](https://github.com/hashicorp/terraform/issues/968))
  * provider/aws: `aws_route_table` can specify propagating VGWs ([#1516](https://github.com/hashicorp/terraform/issues/1516))
  * provider/aws: `aws_route53_record` supports weighted sets ([#1578](https://github.com/hashicorp/terraform/issues/1578))
  * provider/aws: `aws_route53_zone` exports nameservers ([#1525](https://github.com/hashicorp/terraform/issues/1525))
  * provider/aws: `aws_s3_bucket` website support ([#1738](https://github.com/hashicorp/terraform/issues/1738))
  * provider/aws: `aws_security_group` name becomes optional and can be
      automatically set to a unique identifier; this helps with
      `create_before_destroy` scenarios ([#1632](https://github.com/hashicorp/terraform/issues/1632))
  * provider/aws: `aws_security_group` description becomes optional with a
      static default value ([#1632](https://github.com/hashicorp/terraform/issues/1632))
  * provider/aws: automatically set the private IP as the SSH address
      if not specified and no public IP is available ([#1623](https://github.com/hashicorp/terraform/issues/1623))
  * provider/aws: `aws_elb` exports `source_security_group` field ([#1708](https://github.com/hashicorp/terraform/issues/1708))
  * provider/aws: `aws_route53_record` supports alias targeting ([#1775](https://github.com/hashicorp/terraform/issues/1775))
  * provider/aws: Remove default AWS egress rule for newly created Security Groups ([#1765](https://github.com/hashicorp/terraform/issues/1765))
  * provider/consul: add `scheme` configuration argument ([#1838](https://github.com/hashicorp/terraform/issues/1838))
  * provider/docker: `docker_container` can specify links ([#1564](https://github.com/hashicorp/terraform/issues/1564))
  * provider/google: `resource_compute_disk` supports snapshots ([#1426](https://github.com/hashicorp/terraform/issues/1426))
  * provider/google: `resource_compute_instance` supports specifying the
      device name ([#1426](https://github.com/hashicorp/terraform/issues/1426))
  * provider/openstack: Floating IP support for LBaaS ([#1550](https://github.com/hashicorp/terraform/issues/1550))
  * provider/openstack: Add AZ to `openstack_blockstorage_volume_v1` ([#1726](https://github.com/hashicorp/terraform/issues/1726))

BUG FIXES:

  * core: Fix graph cycle issues surrounding modules ([#1582](https://github.com/hashicorp/terraform/issues/1582)) ([#1637](https://github.com/hashicorp/terraform/issues/1637))
  * core: math on arbitrary variables works if first operand isn't a
      numeric primitive. ([#1381](https://github.com/hashicorp/terraform/issues/1381))
  * core: avoid unnecessary cycles by pruning tainted destroys from
      graph if there are no tainted resources ([#1475](https://github.com/hashicorp/terraform/issues/1475))
  * core: fix issue where destroy nodes weren't pruned in specific
      edge cases around matching prefixes, which could cause cycles ([#1527](https://github.com/hashicorp/terraform/issues/1527))
  * core: fix issue causing diff mismatch errors in certain scenarios during
      resource replacement ([#1515](https://github.com/hashicorp/terraform/issues/1515))
  * core: dependencies on resources with a different index work when
      count > 1 ([#1540](https://github.com/hashicorp/terraform/issues/1540))
  * core: don't panic if variable default type is invalid ([#1344](https://github.com/hashicorp/terraform/issues/1344))
  * core: fix perpetual diff issue for computed maps that are empty ([#1607](https://github.com/hashicorp/terraform/issues/1607))
  * core: validation added to check for `self` variables in modules ([#1609](https://github.com/hashicorp/terraform/issues/1609))
  * core: fix edge case where validation didn't pick up unknown fields
      if the value was computed ([#1507](https://github.com/hashicorp/terraform/issues/1507))
  * core: Fix issue where values in sets on resources couldn't contain
      hyphens. ([#1641](https://github.com/hashicorp/terraform/issues/1641))
  * core: Outputs removed from the config are removed from the state ([#1714](https://github.com/hashicorp/terraform/issues/1714))
  * core: Validate against the worst-case graph during plan phase to catch cycles
      that would previously only show up during apply ([#1655](https://github.com/hashicorp/terraform/issues/1655))
  * core: Referencing invalid module output in module validates ([#1448](https://github.com/hashicorp/terraform/issues/1448))
  * command: remote states with uppercase types work ([#1356](https://github.com/hashicorp/terraform/issues/1356))
  * provider/aws: Support `AWS_SECURITY_TOKEN` env var again ([#1785](https://github.com/hashicorp/terraform/issues/1785))
  * provider/aws: Don't save "instance" for EIP if association fails ([#1776](https://github.com/hashicorp/terraform/issues/1776))
  * provider/aws: launch configuration ID set after create success ([#1518](https://github.com/hashicorp/terraform/issues/1518))
  * provider/aws: Fixed an issue with creating ELBs without any tags ([#1580](https://github.com/hashicorp/terraform/issues/1580))
  * provider/aws: Fix issue in Security Groups with empty IPRanges ([#1612](https://github.com/hashicorp/terraform/issues/1612))
  * provider/aws: manually deleted S3 buckets are refreshed properly ([#1574](https://github.com/hashicorp/terraform/issues/1574))
  * provider/aws: only check for EIP allocation ID in VPC ([#1555](https://github.com/hashicorp/terraform/issues/1555))
  * provider/aws: raw protocol numbers work in `aws_network_acl` ([#1435](https://github.com/hashicorp/terraform/issues/1435))
  * provider/aws: Block devices can be encrypted ([#1718](https://github.com/hashicorp/terraform/issues/1718))
  * provider/aws: ASG health check grace period can be updated in-place ([#1682](https://github.com/hashicorp/terraform/issues/1682))
  * provider/aws: ELB security groups can be updated in-place ([#1662](https://github.com/hashicorp/terraform/issues/1662))
  * provider/aws: `aws_main_route_table
