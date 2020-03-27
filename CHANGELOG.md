## 0.13.0 (Unreleased)

BREAKING CHANGES:

* backend/oss: Changes to the TableStore schema now require a primary key named `LockID` of type `String` [GH-24149]
* command/0.12upgrade: this command has been replaced with a deprecation notice directing users to install terraform v0.12 to run `terraform 0.12upgrade`.  [GH-24403]
* command/import: remove the deprecated `-provider` command line argument [GH-24090]
* command/import: fixed a bug where the `import` command was not properly attaching the configured provider for a resource to be imported, making the `-provider` command line argument unnecessary. [GH-22862]
* config: Inside `provisioner` blocks that have `when = destroy` set, and inside any `connection` blocks that are used by such `provisioner` blocks, it is now an error to refer to any objects other than `self`, `count`, or `each` [GH-24083]
* config: The `merge` function now returns more precise type information, making it usable for values passed to `for_each` [GH-24032]
* The official MacOS builds of Terraform CLI are no longer compatible with Mac OS 10.10 Yosemite; Terraform now requires at least Mac OS 10.11 El Capitan. Terraform 0.13 is the last major release that will support 10.11 El Capitan, so if you are upgrading your OS we recommend upgrading to Mac OS 10.12 Sierra or later.
* The official FreeBSD builds of Terraform CLI are no longer compatible with FreeBSD 10.x, which has reached end-of-life. Terraform now requires FreeBSD 11.2 or later.

ENHANCEMENTS:
* config: `templatefile` function will now return a helpful error message if a given variable has an invalid name, rather than relying on a syntax error in the template parsing itself. [GH-24184]
* config: The configuration language now uses Unicode 12.0 character tables for certain Unicode-version-sensitive operations on strings, such as the `upper` and `lower` functions. Those working with strings containing new characters introduced since Unicode 9.0 may see small differences in behavior as a result of these table updates.
* core: added support for passing metadata from modules to providers using HCL [GH-22583]
* Terraform CLI now supports TLS 1.3 and supports Ed25519 certificates when making outgoing connections to remote TLS servers. While both of these changes are backwards compatible in principle, certain legacy TLS server implementations can reportedly encounter problems when attempting to negotiate TLS 1.3. (These changes affects only requests made by Terraform CLI itself, such as to module registries or backends. Provider plugins have separate TLS implementations that will gain these features on a separate release schedule.)
* On Unix systems where `use-vc` is set in `resolv.conf`, Terraform will now use TCP for DNS resolution. We don't expect this to cause any problem for most users, but if you find you are seeing DNS resolution failures after upgrading please verify that you can either reach your configured nameservers using TCP or that your resolver configuration does not include the `use-vc` directive.

BUG FIXES:
* backend/oss: Allow locking of multiple different state files [GH-24149]
* cli: Fix `terraform state mv` to correctly set the resource each mode based on the target address [GH-24254]
* cli: The `terraform plan` command (and the implied plan run by `terraform apply` with no arguments) will now print any warnings that were generated even if there are no changes to be made. [GH-24095]
* command/show (json output): fix inconsistency in resource addresses between plan and prior state output [GH-24256]
* core: Instances are now destroyed only using their stored state, removing many cycle errors [GH-24083]
* lang: Fix non-string key panics in map function [GH-24277]

---
For information on prior major releases, see their changelogs:

* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
