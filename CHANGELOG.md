## 1.1.4 (January 19, 2022)

BUG FIXES:

* config: Non-nullable variables with null inputs were not given default values when checking validation statements ([#30330](https://github.com/hashicorp/terraform/issues/30330))
* config: Terraform will no longer incorrectly report "Cross-package move statement" when an external package has changed a resource from no `count` to using `count`, or vice-versa. ([#30333](https://github.com/hashicorp/terraform/issues/30333))

## 1.1.3 (January 06, 2022)

BUG FIXES:

* `terraform init`: Will now remove from [the dependency lock file](https://www.terraform.io/language/files/dependency-lock) entries for providers not used in the current configuration. Previously it would leave formerly-used providers behind in the lock file, leading to "missing or corrupted provider plugins" errors when other commands verified the consistency of the installed plugins against the locked plugins. ([#30192](https://github.com/hashicorp/terraform/issues/30192))
* config: Fix panic when encountering an invalid provider block within a module ([#30095](https://github.com/hashicorp/terraform/issues/30095))
* config: Fix cycle error when the index of a module containing move statements is changed ([#30232](https://github.com/hashicorp/terraform/issues/30232))
* config: Fix inconsistent ordering with nested move operations ([#30253](https://github.com/hashicorp/terraform/issues/30253))
* config: Fix `moved` block refactoring to include nested modules ([#30233](https://github.com/hashicorp/terraform/issues/30233))
* functions: Redact sensitive values from function call error messages ([#30067](https://github.com/hashicorp/terraform/issues/30067))
* `terraform show`: Disable plan state lineage checks, ensuring that we can show plan files which were generated against non-default state files ([#30205](https://github.com/hashicorp/terraform/issues/30205))

## 1.1.2 (December 17, 2021)

**If you are using Terraform CLI v1.1.0 or v1.1.1, please upgrade to this new version as soon as possible.**

Terraform CLI v1.1.0 and v1.1.1 both have a bug where a failure to construct the apply-time graph can cause Terraform to incorrectly report success and save an empty state, effectively "forgetting" all existing infrastructure. Although configurations that already worked on previous releases should not encounter this problem, it's possible that incorrect _future_ configuration changes would trigger this behavior during the apply step.

BUG FIXES:

* config: Fix panic when using `-target` in combination with `moved` blocks within modules ([#30189](https://github.com/hashicorp/terraform/issues/30189))
* core: Fix condition which could lead to an empty state being written when there is a failure building the apply graph ([#30199](https://github.com/hashicorp/terraform/issues/30199))

## 1.1.1 (December 15, 2021)

BUG FIXES:

* core: Fix crash with orphaned module instance due to changed `count` or `for_each` value ([#30151](https://github.com/hashicorp/terraform/issues/30151))
* core: Fix regression where some expressions failed during validation when referencing resources expanded with `count` or `for_each` ([#30171](https://github.com/hashicorp/terraform/issues/30171))

## 1.1.0 (December 08, 2021)

Terraform v1.1.0 is a new minor release, containing some new features and some bug fixes whose scope was too large for inclusion in a patch release.

NEW FEATURES:

* `moved` blocks for refactoring within modules: Module authors can now record in module source code whenever they've changed the address of a resource or resource instance, and then during planning Terraform will automatically migrate existing objects in the state to new addresses.

    This therefore avoids the need for users of a shared module to manually run `terraform state mv` after upgrading to a version of the module, as long as the change is expressible as static configuration. However, `terraform state mv` will remain available for use in more complex migration situations that are not well-suited to declarative configuration.
* A new `cloud` block in the `terraform` settings block introduces a native Terraform Cloud integration for the [CLI-driven run workflow](https://www.terraform.io/docs/cloud/run/cli.html).

    The Cloud integration includes several enhancements, including per-run variable support using the `-var` flag, the ability to map Terraform Cloud workspaces to the current configuration via [Workspace Tags](https://www.terraform.io/docs/cloud/api/workspaces.html#get-tags), and an improved user experience for Terraform Cloud and Enterprise users with actionable error messages and prompts.
* `terraform plan` and `terraform apply` both now include additional annotations for resource instances planned for deletion to explain why Terraform has proposed that action.

    For example, if you change the `count` argument for a resource to a lower number then Terraform will now mention that as part of proposing to destroy any existing objects that exceed the new count.

UPGRADE NOTES:

This release is covered by the [Terraform v1.0 Compatibility Promises](https://www.terraform.io/docs/language/v1-compatibility-promises.html), but does include some changes permitted within those promises as described below.

* Terraform on macOS now requires macOS 10.13 High Sierra or later; Older macOS versions are no longer supported.
* The `terraform graph` command no longer supports `-type=validate` and `-type=eval` options. The validate graph is always the same as the plan graph anyway, and the "eval" graph was just an implementation detail of the `terraform console` command. The default behavior of creating a plan graph should be a reasonable replacement for both of the removed graph modes. (Please note that `terraform graph` is not covered by the Terraform v1.0 compatibility promises, because its behavior inherently exposes Terraform Core implementation details, so we recommend it only for interactive debugging tasks and not for use in automation.)
* `terraform apply` with a previously-saved plan file will now verify that the provider plugin packages used to create the plan fully match the ones used during apply, using the same checksum scheme that Terraform normally uses for the dependency lock file. Previously Terraform was checking consistency of plugins from a plan file using a legacy mechanism which covered only the main plugin executable, not any other files that might be distributed alongside in the plugin package.

    This additional check should not affect typical plugins that conform to the expectation that a plugin package's contents are immutable once released, but may affect a hypothetical in-house plugin that intentionally modifies extra files in its package directory somehow between plan and apply. If you have such a plugin, you'll need to change its approach to store those files in some other location separate from the package directory. This is a minor compatibility break motivated by increasing the assurance that plugins have not been inadvertently or maliciously modified between plan and apply.
* `terraform state mv` will now error when legacy `-backup` or `-backup-out` options are used without the `-state` option on non-local backends. These options operate on a local state file only. Previously, these options were accepted but ignored silently when used with non-local backends. 
* In the AzureRM backend, the new opt-in option `use_microsoft_graph` switches to using MSAL authentication tokens and Microsoft Graph rather than using ADAL tokens and Azure Active Directory Graph, which is now [deprecated by Microsoft](https://docs.microsoft.com/en-us/graph/migrate-azure-ad-graph-faq). The new mode will become the default in Terraform v1.2, so please plan to migrate to using this setting and test with your own Azure AD tenant prior to the Terraform v1.2 release.

ENHANCEMENTS:

* config: Terraform now checks the syntax of and normalizes module source addresses (the `source` argument in `module` blocks) during configuration decoding rather than only at module installation time. This is largely just an internal refactoring, but a visible benefit of this change is that the `terraform init` messages about module downloading will now show the canonical module package address Terraform is downloading from, after interpreting the special shorthands for common cases like GitHub URLs. ([#28854](https://github.com/hashicorp/terraform/issues/28854))
* config: Variables can now be declared as "nullable", which defines whether a variable can be null within a module. Setting `nullable = false` ensures that a variable value will never be `null`, and may instead take on the variable's default value if the caller sets it explicitly to `null`. ([#29832](https://github.com/hashicorp/terraform/issues/29832))
* `terraform plan` and `terraform apply`: When Terraform plans to destroy a resource instance due to it no longer being declared in the configuration, the proposed plan output will now include a note hinting at what situation prompted that proposal, so you can more easily see what configuration change might avoid the object being destroyed. ([#29637](https://github.com/hashicorp/terraform/pull/29637))
* `terraform plan` and `terraform apply`: Terraform will now report explicitly in the UI if it automatically moves a resource instance to a new address as a result of adding or removing the `count` argument from an existing resource. For example, if you previously had `resource "aws_subnet" "example"` _without_ `count`, you might have `aws_subnet.example` already bound to a remote object in your state. If you add `count = 1` to that resource then Terraform would previously silently rebind the object to `aws_subnet.example[0]` as part of planning, whereas now Terraform will mention that it did so explicitly in the plan description. ([#29605](https://github.com/hashicorp/terraform/issues/29605))
* `terraform workspace delete`: will now allow deleting a workspace whose state contains only data resource instances and output values, without running `terraform destroy` first. Previously the presence of data resources would require using `-force` to override the safety check guarding against accidentally forgetting about remote objects, but a data resource is not responsible for the management of its associated remote object(s) and so there's no reason to require explicit deletion. ([#29754](https://github.com/hashicorp/terraform/issues/29754))
* `terraform validate`: Terraform now uses precise type information for resources during config validation, allowing more problems to be caught that that step rather than only during the planning step. ([#29862](https://github.com/hashicorp/terraform/issues/29862))
* provisioner/remote-exec and provisioner/file: When using SSH agent authentication mode on Windows, Terraform can now detect and use [the Windows 10 built-in OpenSSH Client](https://devblogs.microsoft.com/powershell/using-the-openssh-beta-in-windows-10-fall-creators-update-and-windows-server-1709/)'s SSH Agent, when available, in addition to the existing support for the third-party solution [Pageant](https://documentation.help/PuTTY/pageant.html) that was already supported. ([#29747](https://github.com/hashicorp/terraform/issues/29747))
* cli: `terraform state mv` will now return an error for `-backup` or `-backup-out` options used without the `-state` option, unless the working directory is initialized to use the local backend. Previously Terraform would silently ignore those options, since they are applicable only to the local backend. ([#27908](https://github.com/hashicorp/terraform/issues/27908))
* `terraform console`: now has a new `type()` function, available only in the interactive console, for inspecting the exact type of a particular value as an aid to debugging. ([#28501](https://github.com/hashicorp/terraform/issues/28501))

BUG FIXES:

* config: `ignore_changes = all` now works in override files. ([#29849](https://github.com/hashicorp/terraform/issues/29849))
* config: Upgrading an unknown single value to a list using a splat expression now correctly returns an unknown value and type. Previously it would sometimes "overpromise" a particular return type, leading to an inconsistency error during the apply step. ([#30062](https://github.com/hashicorp/terraform/issues/30062))
* config: Terraform is now more precise in its detection of data resources that must be deferred to the apply step due to their `depends_on` arguments referring to not-yet-converged managed resources. ([#29682](https://github.com/hashicorp/terraform/issues/29682))
* config: `ignore_changes` can no longer cause a null map to be converted to an empty map, which would otherwise potentially cause surprising side-effects in provider logic. ([#29928](https://github.com/hashicorp/terraform/issues/29928))
* core: Provider configuration obtained from interactive prompts will now be merged properly with settings given in the configuration. Previously this merging was incorrect in some cases. ([#29000](https://github.com/hashicorp/terraform/issues/29000))
* `terraform plan`: Improved rendering of changes inside attributes that accept lists, sets, or maps of nested object types. ([#29827](https://github.com/hashicorp/terraform/issues/29827), [#29983](https://github.com/hashicorp/terraform/issues/29983), [#29986](https://github.com/terraform/issues/29986))
* `terraform apply`: Will no longer try to apply a stale plan that was generated against an originally-empty state. Previously this was an unintended exception to the rule that a plan can only be applied to the state snapshot it was generated against. ([#29755](https://github.com/hashicorp/terraform/issues/29755))
* `terraform show -json`: Attributes that are declared as using the legacy [Attributes as Blocks](https://www.terraform.io/docs/language/attr-as-blocks.html) behavior are now represented more faithfully in the JSON plan output. ([#29522](https://github.com/hashicorp/terraform/issues/29522))
* `terraform init`: Will now update the backend configuration hash value at a more approprimate time, to ensure properly restarting a backend migration process that failed on the first attempt. ([#29860](https://github.com/hashicorp/terraform/issues/29860))
* backend/oss: Flatten `assume_role` block arguments, so that they are more compatible with the `terraform_remote_state` data source. ([#29307](https://github.com/hashicorp/terraform/issues/29307))

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
