## 0.13.0-beta2 (UNRELEASED)

NOTES:

* backend/s3: Deprecated `lock_table`, `skip_get_ec2_platforms`, and `skip_requesting_account_id` arguments have been removed [GH-25134]
* backend/s3: Credential ordering has changed from static, environment, shared credentials, EC2 metadata, default AWS Go SDK (shared configuration, web identity, ECS, EC2 Metadata) to static, environment, shared credentials, default AWS Go SDK (shared configuration, web identity, ECS, EC2 Metadata) [GH-25134]
* backend/s3: The `AWS_METADATA_TIMEOUT` environment variable no longer has any effect as we now depend on the default AWS Go SDK EC2 Metadata client timeout of one second with two retries [GH-25134]
* Removed unused targets from Makefile. If you were previously using `make dev` or `make quickdev`, replace that usage with `go install` [GH-25146]

ENHANCEMENTS:

* backend/kubernetes: New `kubernetes` remote state storage backend [GH-19525]
* backend/s3: Always enable shared configuration file support (no longer require `AWS_SDK_LOAD_CONFIG` environment variable) [GH-25134]
* backend/s3: Automatically expand `~` prefix for home directories in `shared_credentials_file` argument [GH-25134]
* backend/s3: Add `assume_role_duration_seconds`, `assume_role_policy_arns`, `assume_role_tags`, and `assume_role_transitive_tag_keys` arguments [GH-25134]
* command/providers: Show providers in a tree of modules requiring them, along with a list of providers required by state [GH-25190]

BUG FIXES:

* addrs: detect builtin "terraform" provider in legacy state [GH-25154]
* backend/remote: do not panic if PrepareConfig or Configure receive null values (can occur when the user cancels the init command) [GH-25135]
* backend/s3: Ensure configured profile is used [GH-25134]
* backend/s3: Ensure configured STS endpoint is used during AssumeRole API calls [GH-25134]
* backend/s3: Prefer AWS shared configuration over EC2 metadata credentials by default [GH-25134]
* backend/s3: Prefer ECS credentials over EC2 metadata credentials by default [GH-25134]
* backend/s3: Remove hardcoded AWS Provider messaging [GH-25134]
* command/0.13upgrade: Fix `0.13upgrade` usage help text to include options [GH-25127]
* command/0.13upgrade: Do not add source for builtin provider [GH-25215]
* command/apply: Fix bug which caused Terraform to silently exit on Windows when using absolute plan path [GH-25233]
* command/init: Fix bug which caused the default local plugindir to be omitted as a provider source location [GH-25214]
* command/format: Fix bug which caused some diagnostics to print empty source lines [GH-25156]
* command/version: add -json flag for machine-parsable version output [GH-25252]
* config: Function argument expansion with `...` will no longer incorrectly return "Invalid expanding argument value" in situations where the expanding argument type isn't known yet. [GH-25216]
* config: Fix crash in validation with non-ascii characters [GH-25144]
* config: Don't panic if version constraint syntax isn't accepted by new version constraint parser [GH-25223]
* core: Fix crash with multiple nested modules [GH-25176]
* core: Fix panic when importing with modules [GH-25208]
* core: Allow targeting with expanded module addresses [GH-25206]

## 0.13.0-beta1 (June 03, 2020)

NEW FEATURES:

* **`count` and `for_each` for modules**: Similar to the arguments of the same name in `resource` and `data` blocks, these create multiple instances of a module from a single `module` block. ([#24461](https://github.com/hashicorp/terraform/issues/24461))
* **`depends_on` for modules**: Modules can now use the `depends_on` argument to ensure that all module resource changes will be applied after any changes to the `depends_on` targets have been applied. ([#25005](https://github.com/hashicorp/terraform/issues/25005))
* **Automatic installation of third-party providers**: Terraform now supports a decentralized namespace for providers, allowing for automatic installation of community providers from third-party namespaces in the public registry and from private registries. (More details will be added about this prior to release.)
* **Custom validation rules for input variables**: A new [`validation` block type](https://www.terraform.io/docs/configuration/variables.html#custom-validation-rules) inside `variable` blocks allows module authors to define validation rules at the public interface into a module, so that errors in the calling configuration can be reported in the caller's context rather than inside the implementation details of the module. ([#25054](https://github.com/hashicorp/terraform/issues/25054))

BREAKING CHANGES:

* As part of implementing a new decentralized namespace for providers, Terraform now requires an explicit `source` specification for any provider that is not in the "hashicorp" namespace in the main public registry. (More details will be added about this prior to release, including links to upgrade steps.) ([#24477](https://github.com/hashicorp/terraform/issues/24477))
* backend/oss: Changes to the TableStore schema now require a primary key named `LockID` of type `String` ([#24149](https://github.com/hashicorp/terraform/issues/24149))
* command/0.12upgrade: this command has been replaced with a deprecation notice directing users to install terraform v0.12 to run `terraform 0.12upgrade`.  ([#24403](https://github.com/hashicorp/terraform/issues/24403))
* command/import: remove the deprecated `-provider` command line argument ([#24090](https://github.com/hashicorp/terraform/issues/24090))
* command/import: fixed a bug where the `import` command was not properly attaching the configured provider for a resource to be imported, making the `-provider` command line argument unnecessary. ([#22862](https://github.com/hashicorp/terraform/issues/22862))
* command/providers: the output of this command is now a flat list that does not display providers per module. ([#24634](https://github.com/hashicorp/terraform/issues/24634))
* config: Inside `provisioner` blocks that have `when = destroy` set, and inside any `connection` blocks that are used by such `provisioner` blocks, it is now an error to refer to any objects other than `self`, `count`, or `each` ([#24083](https://github.com/hashicorp/terraform/issues/24083))
* configs: At most one `terraform.required_providers` block is permitted per module ([#24763](https://github.com/hashicorp/terraform/issues/24763))
* The official MacOS builds of Terraform CLI are no longer compatible with Mac OS 10.10 Yosemite; Terraform now requires at least Mac OS 10.11 El Capitan. Terraform 0.13 is the last major release that will support 10.11 El Capitan, so if you are upgrading your OS we recommend upgrading to Mac OS 10.12 Sierra or later.
* The official FreeBSD builds of Terraform CLI are no longer compatible with FreeBSD 10.x, which has reached end-of-life. Terraform now requires FreeBSD 11.2 or later.

NOTES:

* The `terraform plan` and `terraform apply` command will now detect and report changes to root module outputs as able to be applied even if there are no resource changes in the plan. This will be an improvement in behavior for most users, since it will now be possible to change `output` blocks and use `terraform apply` to apply those changes, but it may require some changes to unusual situations where a root module output value was _intentionally_ changing on every plan, which was not an intended usage pattern and is no longer supported.
* Terraform CLI now supports TLS 1.3 and supports Ed25519 certificates when making outgoing connections to remote TLS servers. While both of these changes are backwards compatible in principle, certain legacy TLS server implementations can reportedly encounter problems when attempting to negotiate TLS 1.3. (These changes affects only requests made by Terraform CLI itself, such as to module registries or backends. Provider plugins have separate TLS implementations that will gain these features on a separate release schedule.)
* On Unix systems where `use-vc` is set in `resolv.conf`, Terraform will now use TCP for DNS resolution. We don't expect this to cause any problem for most users, but if you find you are seeing DNS resolution failures after upgrading please verify that you can either reach your configured nameservers using TCP or that your resolver configuration does not include the `use-vc` directive.
* backend/s3: Region validation now automatically supports the new `af-south-1` (Africa (Cape Town)) region. For AWS operations to work in the new region, the region must be explicitly enabled as outlined in the [AWS Documentation](https://docs.aws.amazon.com/general/latest/gr/rande-manage.html#rande-manage-enable). When the region is not enabled, the Terraform S3 Backend will return errors during credential validation (e.g. `error validating provider credentials: error calling sts:GetCallerIdentity: InvalidClientTokenId: The security token included in the request is invalid`). ([#24744](https://github.com/hashicorp/terraform/issues/24744))

ENHANCEMENTS:

* config: `templatefile` function will now return a helpful error message if a given variable has an invalid name, rather than relying on a syntax error in the template parsing itself. ([#24184](https://github.com/hashicorp/terraform/issues/24184))
* config: The configuration language now uses Unicode 12.0 character tables for certain Unicode-version-sensitive operations on strings, such as the `upper` and `lower` functions. Those working with strings containing new characters introduced since Unicode 9.0 may see small differences in behavior as a result of these table updates.
* cli: When installing providers from the Terraform Registry, Terraform will verify the trust signature for partner providers, and allow for self-signed community providers ([#24617](https://github.com/hashicorp/terraform/issues/24617))
* cli: Display detailed trust signature information when installing providers from the Terraform Registry, including a link to more documentation on different levels of signature ([#24932](https://github.com/hashicorp/terraform/issues/24932))
* cli: It is now possible to optionally specify explicitly which installation methods can be used for different providers, such as forcing a particular provider to be loaded from a particular directory on local disk instead of consulting its origin provider registry. ([#24728](https://github.com/hashicorp/terraform/issues/24728))
* cli: Add state replace-provider subcommand to allow changing the provider source for existing resources ([#24523](https://github.com/hashicorp/terraform/issues/24523))
* cli: The `terraform plan` and `terraform apply` commands now recognize changes to root module outputs as side-effects to be approved and applied. This means you can apply root module output changes using the normal plan and apply workflow. ([#25047](https://github.com/hashicorp/terraform/issues/25047))
* cli: The new `terraform providers mirror` subcommand can automatically construct or update a local filesystem mirror directory containing the providers required for the current configuration. ([#25084](https://github.com/hashicorp/terraform/issues/25084))
* config: The `merge` function now returns more precise type information, making it usable for values passed to `for_each` ([#24032](https://github.com/hashicorp/terraform/issues/24032))
* config: Add "sum" function, which takes a list or set of numbers and returns the sum of all elements ([#24666](https://github.com/hashicorp/terraform/issues/24666))
* config: added support for passing metadata from modules to providers using HCL ([#22583](https://github.com/hashicorp/terraform/issues/22583))
* core: Significant performance enhancements for graph operations, which will help with highly-connected graphs ([#23811](https://github.com/hashicorp/terraform/issues/23811))
* core: Data resources can now be evaluated during plan, allowing `depends_on` to work correctly, and allowing data sources to update immediately when their configuration changes. ([#24904](https://github.com/hashicorp/terraform/issues/24904))
* core: Data resource changes detected during planning will now always be reported in the plan output, to highlight a likely reason for effective configurationc changes elsewhere. Previously only data resources deferred to the apply phase would be shown. ([#24904](https://github.com/hashicorp/terraform/issues/24904))
* backend/azurerm: switching to use the Giovanni Storage SDK to communicate with Azure ([#24669](https://github.com/hashicorp/terraform/issues/24669))
* backend/remote: Now supports `terraform state push -force`. ([#24696](https://github.com/hashicorp/terraform/issues/24696))
* backend/remote: Can now accept `-target` options when creating a plan using _remote operations_, if supported by the target server. (Server-side support for this in Terraform Cloud and Terraform Enterprise will follow in forthcoming releases of each.) ([#24834](https://github.com/hashicorp/terraform/issues/24834))
* backend/s3: Support automatic region validation for `af-south-1` ([#24744](https://github.com/hashicorp/terraform/issues/24744))
* backend/swift auth options match OpenStack provider auth ([#23510](https://github.com/hashicorp/terraform/issues/23510))

BUG FIXES:
* backend/oss: Allow locking of multiple different state files ([#24149](https://github.com/hashicorp/terraform/issues/24149))
* cli: Fix `terraform state mv` to correctly set the resource each mode based on the target address ([#24254](https://github.com/hashicorp/terraform/issues/24254))
* cli: The `terraform plan` command (and the implied plan run by `terraform apply` with no arguments) will now print any warnings that were generated even if there are no changes to be made. ([#24095](https://github.com/hashicorp/terraform/issues/24095))
* cli: When using the `TF_CLI_CONFIG_FILE` environment variable to override where Terraform looks for CLI configuration, Terraform will now ignore the default CLI configuration directory as well as the default CLI configuration file. ([#24728](https://github.com/hashicorp/terraform/issues/24728))
* cli: The `terraform login` command in OAuth2 mode now implements the PKCE OAuth 2 extension more correctly. Previously it was not compliant with all of the details of the specification. ([#24858](https://github.com/hashicorp/terraform/issues/24858))
* command: Fix panic for nil credentials source, which can be caused by an unset HOME environment variable ([#25110](https://github.com/hashicorp/terraform/issues/25110))
* command/fmt: Include source snippets in errors ([#24471](https://github.com/hashicorp/terraform/issues/24471))
* command/format: Fix diagnostic output to show multi-line code snippets correctly ([#24473](https://github.com/hashicorp/terraform/issues/24473))
* command/show (json output): fix inconsistency in resource addresses between plan and prior state output ([#24256](https://github.com/hashicorp/terraform/issues/24256))
* core: Fix race in GetVariableValue ([#24599](https://github.com/hashicorp/terraform/issues/24599))
* core: Instances are now destroyed only using their stored state, removing many cycle errors ([#24083](https://github.com/hashicorp/terraform/issues/24083))
* core: Destroy provisioners should not evaluate for_each expressions ([#24163](https://github.com/hashicorp/terraform/issues/24163))
* funcs/jsonencode: Fix null value handling ([#25078](https://github.com/hashicorp/terraform/issues/25078))
* lang: Fix non-string key panics in map function ([#24277](https://github.com/hashicorp/terraform/issues/24277))
* lang: `substr("abc", 0, 0)` would previously return `"abc"`, despite the length argument being `0`. This has been changed to return an empty string when length is zero. ([#24318](https://github.com/hashicorp/terraform/issues/24318))
* lang: `ceil(1/0)` and `floor(1/0)` would previously return a large integer value, rather than infinity. This has been fixed. ([#21463](https://github.com/hashicorp/terraform/issues/21463))
* lang: Add support for OpenSSH RSA key format to `rsadecrypt` function ([#25112](https://github.com/hashicorp/terraform/issues/25112))
* provisioner/chef: Gracefully handle non-failure (RFC062) exit codes ([#19155](https://github.com/hashicorp/terraform/issues/19155))
* provisioner/habitat: Fix permissions on user.toml ([#24321](https://github.com/hashicorp/terraform/issues/24321))
* provider/terraform: The `terraform_remote_state` data source will no longer attempt to "configure" the selected backend during validation, which means backends will not try to perform remote actions such as verifying credentials during `terraform validate`. Local validation still applies in all cases, and the configuration step will still occur prior to actually reading the remote state in a normal plan/apply operation. ([#24887](https://github.com/hashicorp/terraform/issues/24887))
* vendor: Fix crash when processing malformed JSON ([#24650](https://github.com/hashicorp/terraform/issues/24650))

EXPERIMENTS:

* This release concludes the `variable_validation` [experiment](https://www.terraform.io/docs/configuration/terraform.html#experimental-language-features) that was started in Terraform v0.12.20. If you were participating in the experiment, you should remove the experiment opt-in from your configuration as part of upgrading to Terraform 0.13.

    The experiment received only feedback that can be addressed with backward-compatible future enhancements, so it's been accepted into this release as stable with no changes to its original design so far. We'll consider additional features related to custom validation in future releases after seeing how it's used in real-world modules.

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
