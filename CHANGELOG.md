## 0.13.0 (Unreleased)

NEW FEATURES:

* Terraform now supports a decentralized namespace for providers, allowing for automatic installation of community providers from third-party namespaces in the public registry and from private registries. (More details will be added about this prior to release.)

BREAKING CHANGES:

* As part of implementing a new decentralized namespace for providers, Terraform now requires an explicit `source` specification for any provider that is not in the "hashicorp" namespace in the main public registry. (More details will be added about this prior to release, including links to upgrade steps.) [GH-24477]
* backend/oss: Changes to the TableStore schema now require a primary key named `LockID` of type `String` [GH-24149]
* command/0.12upgrade: this command has been replaced with a deprecation notice directing users to install terraform v0.12 to run `terraform 0.12upgrade`.  [GH-24403]
* command/import: remove the deprecated `-provider` command line argument [GH-24090]
* command/import: fixed a bug where the `import` command was not properly attaching the configured provider for a resource to be imported, making the `-provider` command line argument unnecessary. [GH-22862]
* command/providers: the output of this command is now a flat list that does not display providers per module. [GH-24634]
* config: Inside `provisioner` blocks that have `when = destroy` set, and inside any `connection` blocks that are used by such `provisioner` blocks, it is now an error to refer to any objects other than `self`, `count`, or `each` [GH-24083]
* config: The `merge` function now returns more precise type information, making it usable for values passed to `for_each` [GH-24032]
* configs: At most one `terraform.required_providers` block is permitted per module [GH-24763]
* The official MacOS builds of Terraform CLI are no longer compatible with Mac OS 10.10 Yosemite; Terraform now requires at least Mac OS 10.11 El Capitan. Terraform 0.13 is the last major release that will support 10.11 El Capitan, so if you are upgrading your OS we recommend upgrading to Mac OS 10.12 Sierra or later.
* The official FreeBSD builds of Terraform CLI are no longer compatible with FreeBSD 10.x, which has reached end-of-life. Terraform now requires FreeBSD 11.2 or later.

NOTES:

* backend/s3: Region validation now automatically supports the new `af-south-1` (Africa (Cape Town)) region. For AWS operations to work in the new region, the region must be explicitly enabled as outlined in the [AWS Documentation](https://docs.aws.amazon.com/general/latest/gr/rande-manage.html#rande-manage-enable). When the region is not enabled, the Terraform S3 Backend will return errors during credential validation (e.g. `error validating provider credentials: error calling sts:GetCallerIdentity: InvalidClientTokenId: The security token included in the request is invalid`). [GH-24744]

ENHANCEMENTS:

* backend/s3: Support automatic region validation for `af-south-1` [GH-24744]
* config: `templatefile` function will now return a helpful error message if a given variable has an invalid name, rather than relying on a syntax error in the template parsing itself. [GH-24184]
* config: The configuration language now uses Unicode 12.0 character tables for certain Unicode-version-sensitive operations on strings, such as the `upper` and `lower` functions. Those working with strings containing new characters introduced since Unicode 9.0 may see small differences in behavior as a result of these table updates.
* core: added support for passing metadata from modules to providers using HCL [GH-22583]
* core: significant performance enhancements for graph operations, which will help with highly-connected graphs [GH-23811]
* command/state: Add state replace-provider subcommand to allow changing the provider source for existing resources [GH-24523]
* lang: Add "sum" function, which takes a list or set of numbers and returns the sum of all elements [GH-24666]
* cli: When installing providers from the Terraform Registry, Terraform will verify the trust signature for partner providers, and allow for self-signed community providers [GH-24617]
* cli: It is now possible to optionally specify explicitly which installation methods can be used for different providers, such as forcing a particular provider to be loaded from a particular directory on local disk instead of consulting its origin provider registry. [GH-24728]
* Terraform CLI now supports TLS 1.3 and supports Ed25519 certificates when making outgoing connections to remote TLS servers. While both of these changes are backwards compatible in principle, certain legacy TLS server implementations can reportedly encounter problems when attempting to negotiate TLS 1.3. (These changes affects only requests made by Terraform CLI itself, such as to module registries or backends. Provider plugins have separate TLS implementations that will gain these features on a separate release schedule.)
* On Unix systems where `use-vc` is set in `resolv.conf`, Terraform will now use TCP for DNS resolution. We don't expect this to cause any problem for most users, but if you find you are seeing DNS resolution failures after upgrading please verify that you can either reach your configured nameservers using TCP or that your resolver configuration does not include the `use-vc` directive.
* backend/swift auth options match OpenStack provider auth [GH-23510]
* backend/remote: Add support for force push to remote backend [GH-24696]

BUG FIXES:
* backend/oss: Allow locking of multiple different state files [GH-24149]
* cli: Fix `terraform state mv` to correctly set the resource each mode based on the target address [GH-24254]
* cli: The `terraform plan` command (and the implied plan run by `terraform apply` with no arguments) will now print any warnings that were generated even if there are no changes to be made. [GH-24095]
* cli: When using the `TF_CLI_CONFIG_FILE` environment variable to override where Terraform looks for CLI configuration, Terraform will now ignore the default CLI configuration directory as well as the default CLI configuration file. [GH-24728]
* cli: The `terraform login` command in OAuth2 mode now implements the PKCE OAuth 2 extension more correctly. Previously it was not compliant with all of the details of the specification. [GH-24858]
* command/fmt: Include source snippets in errors [GH-24471]
* command/format: Fix diagnostic output to show multi-line code snippets correctly [GH-24473]
* command/show (json output): fix inconsistency in resource addresses between plan and prior state output [GH-24256]
* core: Fix race in GetVariableValue [GH-24599]
* core: Instances are now destroyed only using their stored state, removing many cycle errors [GH-24083]
* core: Destroy provisioners should not evaluate for_each expressions [GH-24163]
* lang: Fix non-string key panics in map function [GH-24277]
* lang: `substr("abc", 0, 0)` would previously return `"abc"`, despite the length argument being `0`. This has been changed to return an empty string when length is zero. [GH-24318]
* lang: `ceil(1/0)` and `floor(1/0)` would previously return a large integer value, rather than infinity. This has been fixed. [GH-21463]
* provisioner/chef: Gracefully handle non-failure (RFC062) exit codes [GH-19155]
* provisioner/habitat: Fix permissions on user.toml [GH-24321]
* provider/terraform: The `terraform_remote_state` data source will no longer attempt to "configure" the selected backend during validation, which means backends will not try to perform remote actions such as verifying credentials during `terraform validate`. Local validation still applies in all cases, and the configuration step will still occur prior to actually reading the remote state in a normal plan/apply operation. [GH-24887]
* vendor: Fix crash when processing malformed JSON [GH-24650]

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
