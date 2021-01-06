## 0.13.7 (Unreleased)

## 0.13.6 (January 06, 2021)

UPGRADE NOTES:

* The builtin provider's `terraform_remote_state` data source no longer enforces Terraform version checks on the remote state file. This allows Terraform 0.13.6 to access remote state from future Terraform versions, up until a future incompatible state file version upgrade is required. ([#26692](https://github.com/hashicorp/terraform/issues/26692))

BUG FIXES:

* init: setting `-get-plugins` to `false` will now cause a warning, as this flag has been a no-op since 0.13.0 and usage is better served through using `provider_installation` blocks ([#27092](https://github.com/hashicorp/terraform/issues/27092))


## 0.13.5 (October 21, 2020)

BUG FIXES:
* terraform: fix issue where the provider configuration was not properly attached to the configured provider source address by localname ([#26567](https://github.com/hashicorp/terraform/issues/26567))
* core: fix a performance issue when a resource contains a very large and deeply nested schema ([#26577](https://github.com/hashicorp/terraform/issues/26577))
* backend/azurerm: fix an issue when using the metadata host to lookup environments ([#26463](https://github.com/hashicorp/terraform/issues/26463))

## 0.13.4 (September 30, 2020)

UPGRADE NOTES:

* The built-in vendor (third-party) provisioners, which include `habitat`, `puppet`, `chef`, and `salt-masterless` are now deprecated and will be removed in a future version of Terraform. More information [on Discuss](https://discuss.hashicorp.com/t/notice-terraform-to-begin-deprecation-of-vendor-tool-specific-provisioners-starting-in-terraform-0-13-4/13997).
* Deprecated interpolation-only expressions are detected in more contexts in addition to resources and provider configurations. Module calls, data sources, outputs, and locals are now also covered. Terraform also detects interpolation-only expressions in complex values such as lists and objects. An expression like `"${foo}"` should be rewritten as just `foo`. ([#27272](https://github.com/hashicorp/terraform/issues/27272), [#26334](https://github.com/hashicorp/terraform/issues/26334))

BUG FIXES:

* command: Include schemas from required but unused providers in the output of `terraform providers schema`. This allows development tools such as the Terraform language server to offer autocompletion for the first resource for a given provider. ([#26318](https://github.com/hashicorp/terraform/issues/26318))
* core: create_before_destroy status is now updated in the state during refresh ([#26343](https://github.com/hashicorp/terraform/issues/26343))
* core: data sources using `depends_on`, either directly or through their modules, are no longer are forced to wait until apply by other planned data source reads ([#26375](https://github.com/hashicorp/terraform/issues/26375))

## 0.13.3 (September 16, 2020)

BUG FIXES:

* build: fix crash with terraform binary on openBSD ([#26250](https://github.com/hashicorp/terraform/issues/26250))
* core: prevent create_before_destroy cycles by not connecting module close nodes to resource instance destroy nodes ([#26186](https://github.com/hashicorp/terraform/issues/26186))
* core: fix error where plan action changes from CreateThenDelete to DeleteThenCreate ([#26192](https://github.com/hashicorp/terraform/issues/26192))
* core: fix Cycle when create_before_destroy status wasn't checked from state ([#26263](https://github.com/hashicorp/terraform/issues/26263))
* core: fix "inconsistent final plan" error when changing the number of referenced resources to 0 ([#26264](https://github.com/hashicorp/terraform/issues/26264))
* states/remote: fix `state push -force` to work for all backends ([#26190](https://github.com/hashicorp/terraform/issues/26190))

## 0.13.2 (September 02, 2020)

NEW FEATURES:

* **Network-based Mirrors for [Provider Installation](https://www.terraform.io/docs/commands/cli-config.html#provider-installation)**: As an addition to the existing capability of "mirroring" providers into the local filesystem, a network mirror allows publishing copies of providers on an HTTP server and using that as an alternative source for provider packages, for situations where directly accessing the origin registries is impossible or undesirable. ([#25999](https://github.com/hashicorp/terraform/issues/25999))

ENHANCEMENTS:

* backend/http: add support for configuration by environment variable. ([#25439](https://github.com/hashicorp/terraform/issues/25439))
* command: Add support for provider redirects to `0.13upgrade`. If a provider in the Terraform Registry has moved to a new namespace, the `0.13upgrade` subcommand now detects this and follows the redirect where possible. ([#26061](https://github.com/hashicorp/terraform/issues/26061))
* command: Improve `init` error diagnostics when encountering what appears to be an in-house provider required by a pre-0.13 state file. Terraform will now display suggested `terraform state replace-provider` commands which will fix this specific problem. ([#26066](https://github.com/hashicorp/terraform/issues/26066))

BUG FIXES:

* command: Warn instead of error when the `output` subcommand with no arguments results in no outputs. This aligns the UI to match the 0 exit code in this situation, which is notable but not necessarily an error. ([#26036](https://github.com/hashicorp/terraform/issues/26036))
* terraform: Fix crashing bug when reading data sources during plan with blocks backed by objects, not collections ([#26028](https://github.com/hashicorp/terraform/issues/26028))
* terraform: Fix bug where variables values were asked for twice on the command line and provider input values were asked for but not saved ([#26063](https://github.com/hashicorp/terraform/issues/26063))

## 0.13.1 (August 26, 2020)

ENHANCEMENTS:

* config: `cidrsubnet` and `cidrhost` now support address extensions of more than 32 bits ([#25517](https://github.com/hashicorp/terraform/issues/25517))
* cli: The directories that Terraform searches by default for provider plugins can now be symlinks to directories elsewhere. (This applies only to the top-level directory, not to nested directories inside it.) ([#25692](https://github.com/hashicorp/terraform/issues/25692))
* backend/s3: simplified mock handling and assume role testing ([#25903](https://github.com/hashicorp/terraform/issues/25903))
* backend/s3: support for appending data to the User-Agent request header with the TF_APPEND_USER_AGENT environment variable ([#25903](https://github.com/hashicorp/terraform/issues/25903))

BUG FIXES:
* config: Override files containing `module` blocks can now override the special `providers` argument. ([#25496](https://github.com/hashicorp/terraform/issues/25496))
* cli: The state lock will now be unlocked consistently across both the local and remote backends in the `terraform console` and `terraform import` commands. [[#25454](https://github.com/hashicorp/terraform/issues/25454)] 
* cli: The `-target` option to `terraform plan` and `terraform apply` now correctly handles addresses containing module instance indexes. ([#25760](https://github.com/hashicorp/terraform/issues/25760))
* cli: `terraform state mv` can now move the last resource from a module without panicking. ([#25523](https://github.com/hashicorp/terraform/issues/25523))
* cli: If the output of `terraform version` contains an outdated version notice, this is now printed after the version number and not before. ([#25811](https://github.com/hashicorp/terraform/issues/25811))
* command: Prevent creation of workspaces with invalid names via the `TF_WORKSPACE` environment variable, and allow any existing invalid workspaces to be deleted. ([#25262](https://github.com/hashicorp/terraform/issues/25262))
* command: Fix error when multiple `-no-color` flags are set on the command line. ([#25847](https://github.com/hashicorp/terraform/issues/25847))
* command: Fix backend config override validation, allowing the use of `-backend-config` override files with the enhanced remote backend. ([#25960](https://github.com/hashicorp/terraform/issues/25960))
* core: State snapshots now use a consistent ordering for resources that have the same name across different modules. Previously the ordering was undefined. ([#25498](https://github.com/hashicorp/terraform/issues/25498))
* core: A `dynamic` block producing an unknown number of blocks will no longer incorrectly produce the error "Provider produced inconsistent final plan" when the block type is backed by a set of objects. ([#25662](https://github.com/hashicorp/terraform/issues/25662))
* core: Terraform will now silently drop attributes that appear in the state but are not present in the corresponding resource type schema, on the assumption that those attributes existed in a previous version of the provider and have now been removed. ([#25779](https://github.com/hashicorp/terraform/issues/25779))
* core: The state upgrade logic for handling unqualified provider addresses from Terraform v0.11 and earlier will no longer panic when it encounters references to the built-in `terraform` provider. ([#25861](https://github.com/hashicorp/terraform/issues/25861))
* internal: Clean up provider package download temporary files after installing. ([#25990](https://github.com/hashicorp/terraform/issues/25990))
* terraform: Evaluate module call arguments for `terraform import` even if defaults are given for input variables ([#25890](https://github.com/hashicorp/terraform/issues/25890))
* terraform: Fix misleading Terraform `required_version` constraint diagnostics when multiple `required_version` settings exist in a single module ([#25898](https://github.com/hashicorp/terraform/issues/25898))

## 0.13.0 (August 10, 2020)

> This is a list of changes relative to Terraform v0.12.29. To see the
> incremental changelogs for the v0.13.0 prereleases, see
> [the v0.13.0-rc1 changelog](https://github.com/hashicorp/terraform/blob/v0.13.0-rc1/CHANGELOG.md).

This section contains details about various changes in the v0.13 major release. If you are upgrading from Terraform v0.12, we recommend first referring to [the v0.13 upgrade guide](https://www.terraform.io/upgrade-guides/0-13.html) for information on some common concerns during upgrade and guidance on ways to address them. (The final upgrade guide and the documentation for the new features will be published only when v0.13.0 final is released; until then, some links in this section will be non-functional.)

NEW FEATURES:
* [**`count` and `for_each` for modules**](https://www.terraform.io/docs/configuration/modules.html#multiple-instances-of-a-module): Similar to the arguments of the same name in `resource` and `data` blocks, these create multiple instances of a module from a single `module` block. ([#24461](https://github.com/hashicorp/terraform/issues/24461))

* [**`depends_on` for modules**](https://www.terraform.io/docs/configuration/modules.html#other-meta-arguments): Modules can now use the `depends_on` argument to ensure that all module resource changes will be applied after any changes to the `depends_on` targets have been applied. ([#25005](https://github.com/hashicorp/terraform/issues/25005))

* [**Automatic installation of third-party providers**](https://www.terraform.io/docs/configuration/provider-requirements.html): Terraform now supports a decentralized namespace for providers, allowing for automatic installation of community providers from third-party namespaces in the public registry and from private registries. (More details will be added about this prior to release.)

* [**Custom validation rules for input variables**](https://www.terraform.io/docs/configuration/variables.html#custom-validation-rules): A new `validation` block type inside `variable` blocks allows module authors to define validation rules at the public interface into a module, so that errors in the calling configuration can be reported in the caller's context rather than inside the implementation details of the module. ([#25054](https://github.com/hashicorp/terraform/issues/25054))

* [**New Kubernetes remote state storage backend**](https://www.terraform.io/docs/backends/types/kubernetes.html): This backend stores state snapshots as Kubernetes secrets. ([#19525](https://github.com/hashicorp/terraform/issues/19525))


BREAKING CHANGES:
* As part of introducing a new heirarchical namespace for providers, Terraform now requires an explicit `source` specification for any provider that is not in the "hashicorp" namespace in the main public registry. ([#24477](https://github.com/hashicorp/terraform/issues/24477))

    For more information, including information on the automatic upgrade process, refer to [the v0.13 upgrade guide](https://www.terraform.io/upgrade-guides/0-13.html).

* `terraform import`: the previously-deprecated `-provider` option is now removed. ([#24090](https://github.com/hashicorp/terraform/issues/24090))

    To specify a non-default provider configuration for import, add the `provider` meta-argument to the target `resource` block.
* config: Inside `provisioner` blocks that have `when = destroy` set, and inside any `connection` blocks that are used by such `provisioner` blocks, it is no longer valid to refer to any objects other than `self`, `count`, or `each`. (This was previously deprecated in a v0.12 minor release.) ([#24083](https://github.com/hashicorp/terraform/issues/24083))

    If you are using `null_resource` to define provisioners not attached to a real resource, include any values your provisioners need in the `triggers` map and change the provisioner configuration to refer to those values via `self.triggers`.
* configs: At most one `terraform` `required_providers` block is permitted per module ([#24763](https://github.com/hashicorp/terraform/issues/24763))

    If you previously had multiple `required_providers` blocks in the same module, consolidate their requirements together into a single block.
* The official MacOS builds of Terraform CLI are no longer compatible with Mac OS 10.10 Yosemite; Terraform now requires at least Mac OS 10.11 El Capitan.

    Terraform 0.13 is the last major release that will support 10.11 El Capitan, so if you are upgrading your OS we recommend upgrading to Mac OS 10.12 Sierra or later.
* The official FreeBSD builds of Terraform CLI are no longer compatible with FreeBSD 10.x, which has reached end-of-life. Terraform now requires FreeBSD 11.2 or later.
* backend/oss: The TableStore schema now requires a primary key named `LockID` of type `String`. ([#24149](https://github.com/hashicorp/terraform/issues/24149))
* backend/s3: The previously-deprecated `lock_table`, `skip_get_ec2_platforms`, and `skip_requesting_account_id` arguments are now removed. ([#25134](https://github.com/hashicorp/terraform/issues/25134))
* backend/s3: The credential source preference order now considers EC2 instance profile credentials as lower priority than shared configuration, web identity, and ECS role credentials. ([#25134](https://github.com/hashicorp/terraform/issues/25134))
* backend/s3: The `AWS_METADATA_TIMEOUT` environment variable is no longer used. The timeout is now fixed at one second with two retries. ([#25134](https://github.com/hashicorp/terraform/issues/25134))

NOTES:
* The `terraform plan` and `terraform apply` commands will now detect and report changes to root module outputs as needing to be applied even if there are no resource changes in the plan.

    This is an improvement in behavior for most users, since it will now be possible to change `output` blocks and use `terraform apply` to apply those changes.
    
    If you have a configuration where a root module output value is changing for every plan (for example, by referring to an unstable data source), you will need to remove or change that output value in order to allow convergence on an empty plan. Otherwise, each new plan will propose more changes.
* Terraform CLI now supports TLS 1.3 and supports Ed25519 certificates when making outgoing connections to remote TLS servers.

    While both of these changes are backwards compatible in principle, certain legacy TLS server implementations can reportedly encounter problems when attempting to negotiate TLS 1.3. (These changes affects only requests made by Terraform CLI itself, such as to module registries or backends. Provider plugins have separate TLS implementations that will gain these features on a separate release schedule.)
* On Unix systems where `use-vc` is set in `resolv.conf`, Terraform will now use TCP for DNS resolution.

    We don't expect this to cause any problem for most users, but if you find you are seeing DNS resolution failures after upgrading please verify that you can either reach your configured nameservers using TCP or that your resolver configuration does not include the `use-vc` directive.
* The `terraform 0.12upgrade` command is no longer available. ([#24403](https://github.com/hashicorp/terraform/issues/24403))

    To upgrade from Terraform v0.11, first [upgrade to the latest v0.12 release](https://www.terraform.io/upgrade-guides/0-12.html) and then upgrade to v0.13 from there.

ENHANCEMENTS:
* config: `templatefile` function will now return a helpful error message if a given variable has an invalid name, rather than relying on a syntax error in the template parsing itself. ([#24184](https://github.com/hashicorp/terraform/issues/24184))
* config: The configuration language now uses Unicode 12.0 character tables for certain Unicode-version-sensitive operations on strings, such as the `upper` and `lower` functions. Those working with strings containing new characters introduced since Unicode 9.0 may see small differences in behavior as a result of these table updates.
* config: The new `sum` function takes a list or set of numbers and returns the sum of all elements. ([#24666](https://github.com/hashicorp/terraform/issues/24666))
* config: Modules authored by the same vendor as the main provider they use can now pass metadata to the provider to allow for instrumentation and analytics. ([#22583](https://github.com/hashicorp/terraform/issues/22583))
* cli: The `terraform plan` and `terraform apply` commands now recognize changes to root module outputs as side-effects to be approved and applied. This means you can apply root module output changes using the normal plan and apply workflow. ([#25047](https://github.com/hashicorp/terraform/issues/25047))
* cli: When installing providers from the Terraform Registry, Terraform will verify the trust signature for partner providers, and allow for self-signed community providers. ([#24617](https://github.com/hashicorp/terraform/issues/24617))
* cli: `terraform init` will display detailed trust signature information when installing providers from the Terraform Registry and other provider registries. ([#24932](https://github.com/hashicorp/terraform/issues/24932))
* cli: It is now possible to optionally specify explicitly which installation methods can be used for different providers in the CLI configuration, such as forcing a particular provider to be loaded from a particular directory on local disk instead of consulting its origin provider registry. ([#24728](https://github.com/hashicorp/terraform/issues/24728))
* cli: The new `terraform state replace-provider` subcommand allows changing the selected provider for existing resource instances in the Terraform state. ([#24523](https://github.com/hashicorp/terraform/issues/24523))
* cli: The new `terraform providers mirror` subcommand can automatically construct or update a local filesystem mirror directory containing the providers required for the current configuration. ([#25084](https://github.com/hashicorp/terraform/issues/25084))
* cli: `terraform version -json` now produces machine-readable version information. ([#25252](https://github.com/hashicorp/terraform/issues/25252))
* cli: `terraform import` can now work with provider configurations containing references to other objects, as long as the data in question is already known in the current state. ([#25420](https://github.com/hashicorp/terraform/issues/25420))
* cli: The `terraform state rm` command will now exit with status code 1 if the given resource address does not match any resource instances. ([#22300](https://github.com/hashicorp/terraform/issues/22300))
* cli: The `terraform login` command now requires the full word "yes" to confirm, rather than just "y", for consistency with Terraform's other interactive prompts. ([#25379](https://github.com/hashicorp/terraform/issues/25379))
* core: Several of Terraform's graph operations are now better optimized to support configurations with highly-connected graphs. ([#23811](https://github.com/hashicorp/terraform/issues/23811), [#25544](https://github.com/hashicorp/terraform/issues/25544))
* backend/remote: Now supports `terraform state push -force`. ([#24696](https://github.com/hashicorp/terraform/issues/24696))
* backend/remote: Can now accept `-target` options when creating a plan using _remote operations_, if supported by the target server. (Server-side support for this in Terraform Cloud and Terraform Enterprise will follow in forthcoming releases of each.) ([#24834](https://github.com/hashicorp/terraform/issues/24834))
* backend/azurerm: Now uses the Giovanni Storage SDK to communicate with Azure. ([#24669](https://github.com/hashicorp/terraform/issues/24669))
* backend/s3: The backend will now always consult the shared configuration file, even if the `AWS_SDK_LOAD_CONFIG` environment variable isn't set. That environment variable is now ignored. ([#25134](https://github.com/hashicorp/terraform/issues/25134))
* backend/s3: Region validation now automatically supports the new `af-south-1` (Africa (Cape Town)) region. ([#24744](https://github.com/hashicorp/terraform/issues/24744))

    For AWS operations to work in the new region, you must explicitly enable it as described in [AWS General Reference: Enabling a Region](https://docs.aws.amazon.com/general/latest/gr/rande-manage.html#rande-manage-enable). If you haven't enabled the region, the Terraform S3 Backend will return `InvalidClientTokenId` errors during credential validation.
* backend/s3: A `~/` prefix in the `shared_credentials_file` argument is now expanded to the current user's home directory. ([#25134](https://github.com/hashicorp/terraform/issues/25134))
* backend/s3: The backend has a number of new options for customizing the "assume role" behavior, including controlling the lifetime and access policy of temporary credentials. ([#25134](https://github.com/hashicorp/terraform/issues/25134))
* backend/swift: The authentication options match those of the OpenStack provider. ([#23510](https://github.com/hashicorp/terraform/issues/23510))

BUG FIXES:
* config: The `jsonencode` function can now correctly encode a single null value as the JSON expression `null`. ([#25078](https://github.com/hashicorp/terraform/issues/25078))
* config: The `map` function no longer crashes when incorrectly given a non-string key. ([#24277](https://github.com/hashicorp/terraform/issues/24277))
* config: The `substr` function now correctly returns a zero-length string when given a length of zero, rather than ignoring that argument entirely. ([#24318](https://github.com/hashicorp/terraform/issues/24318))
* config: `ceil(1/0)` and `floor(1/0)` (that is, an infinity as an argument) now return another infinity with the same sign, rather than just a large integer. ([#21463](https://github.com/hashicorp/terraform/issues/21463))
* config: The `rsadecrypt` function now supports the OpenSSH RSA key format. ([#25112](https://github.com/hashicorp/terraform/issues/25112))
* config: The `merge` function now returns more precise type information, making it usable for values passed to `for_each`, and will no longer crash if all of the given maps are empty. ([#24032](https://github.com/hashicorp/terraform/issues/24032), [#25303](https://github.com/hashicorp/terraform/issues/25303))
* vendor: The various set-manipulation functions, like `setunion`, will no longer panic if given an unknown set value ([#25318](https://github.com/hashicorp/terraform/issues/25318))
* config: Fixed a crash with incorrect syntax in `.tf.json` and `.tfvars.json` files. ([#24650](https://github.com/hashicorp/terraform/issues/24650))
* config: The function argument expansion syntax `...` no longer incorrectly fails with "Invalid expanding argument value" in situations where the expanding argument's type will not be known until the apply phase. ([#25216](https://github.com/hashicorp/terraform/issues/25216))
* config: Variable `validation` block error message checks no longer fail when non-ASCII characters are present. ([#25144](https://github.com/hashicorp/terraform/issues/25144))
* cli: The `terraform plan` command (and the implied plan run by `terraform apply` with no arguments) will now print any warnings that were generated even if there are no changes to be made. ([#24095](https://github.com/hashicorp/terraform/issues/24095))
* cli: `terraform state mv` now correctly records the resource's use of either `count` or `for_each` based on the given target address. ([#24254](https://github.com/hashicorp/terraform/issues/24254))
* cli: When using the `TF_CLI_CONFIG_FILE` environment variable to override where Terraform looks for CLI configuration, Terraform will now ignore the default CLI configuration directory as well as the default CLI configuration file. ([#24728](https://github.com/hashicorp/terraform/issues/24728))
* cli: The `terraform login` command in OAuth2 mode now implements the PKCE OAuth 2 extension more correctly. Previously it was not compliant with all of the details of the specification. ([#24858](https://github.com/hashicorp/terraform/issues/24858))
* cli: Fixed a potential crash when the `HOME` environment variable isn't set, causing the native service credentials store to be `nil`. ([#25110](https://github.com/hashicorp/terraform/issues/25110))
* command/fmt: Error messages will now include source code snippets where possible. ([#24471](https://github.com/hashicorp/terraform/issues/24471))
* command/apply: `terraform apply` will no longer silently exit when given an absolute path to a saved plan file on Windows. ([#25233](https://github.com/hashicorp/terraform/issues/25233))
* command/init: `terraform init` will now produce an explicit error message if given a non-directory path for its configuration directory argument, and if a `-backend-config` file has a syntax error. Previously these were silently ignored. ([#25300](https://github.com/hashicorp/terraform/pull/25300), [#25411](https://github.com/hashicorp/terraform/issues/25411))
* command/console:  ([#25442](https://github.com/hashicorp/terraform/issues/25442))
* command/import: The `import` command will now properly attach the configured provider for the target resource based on the configuration, making the `-provider` command line option unnecessary. ([#22862](https://github.com/hashicorp/terraform/issues/22862))
* command/import: The `-allow-missing-config` option now works correctly. It was inadvertently disabled as part of v0.12 refactoring. ([#25352](https://github.com/hashicorp/terraform/issues/25352))
* command/show: Resource addresses are now consistently formatted between the plan and prior state in the `-json` output. ([#24256](https://github.com/hashicorp/terraform/issues/24256))
* core: Fixed a crash related to an unsafe concurrent read and write of a map data structure. ([#24599](https://github.com/hashicorp/terraform/issues/24599))
* core: Instances are now destroyed only using their stored state, without re-evaluating configuration. This avoids a number of dependency cycle problems when "delete" actions are included in a plan. ([#24083](https://github.com/hashicorp/terraform/issues/24083))
* provider/terraform: The `terraform_remote_state` data source will no longer attempt to "configure" the selected backend during validation, which means backends will not try to perform remote actions such as verifying credentials during `terraform validate`. Local validation still applies in all cases, and the configuration step will still occur prior to actually reading the remote state in a normal plan/apply operation. ([#24887](https://github.com/hashicorp/terraform/issues/24887))
* backend/remote: Backend will no longer crash if the user cancels backend initialization at an inopportune time, or if there is a connection error. ([#25135](https://github.com/hashicorp/terraform/issues/25135)) ([#25341](https://github.com/hashicorp/terraform/issues/25341))
* backend/azurerm: The backend will now create a Azure storage snapshot of the previous Terraform state snapshot before writing a new one. ([#24069](https://github.com/hashicorp/terraform/issues/24069))
* backend/s3: Various other minor authentication-related fixes previously made in the AWS provider. ([#25134](https://github.com/hashicorp/terraform/issues/25134))
* backend/oss: Now allows locking of multiple different state files. ([#24149](https://github.com/hashicorp/terraform/issues/24149))
* provisioner/remote-exec: The provisioner will now return an explicit error if the `host` connection argument is an empty string. Previously it would repeatedly attempt to resolve an empty hostname until timeout. ([#24080](https://github.com/hashicorp/terraform/issues/24080))
* provisioner/chef: The provisioner will now gracefully handle non-failure (RFC062) exit codes returned from Chef. ([#19155](https://github.com/hashicorp/terraform/issues/19155))
* provisioner/habitat: The provisioner will no longer generate `user.toml` with world-readable permissions. ([#24321](https://github.com/hashicorp/terraform/issues/24321))
* communicator/winrm: Support a connection timeout for WinRM `connection` blocks. Previously this argument worked for SSH only. ([#25350](https://github.com/hashicorp/terraform/issues/25350))

EXPERIMENTS:
* This release concludes the `variable_validation` [experiment](https://www.terraform.io/docs/configuration/terraform.html#experimental-language-features) that was started in Terraform v0.12.20. If you were participating in the experiment, you should remove the experiment opt-in from your configuration as part of upgrading to Terraform 0.13.

    The experiment received only feedback that can be addressed with backward-compatible future enhancements, so we've included it into this release as stable with no changes to its original design so far. We'll consider additional features related to custom validation in future releases after seeing how it's used in real-world modules.

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
