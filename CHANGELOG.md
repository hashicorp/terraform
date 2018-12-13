## 0.12.0-beta (unreleased)

IMPROVEMENTS:

* plugins: Plugin RPC connection is now authenticated [GH-19629]
* backend/azurerm: Support for authenticating using the Azure CLI [GH-19465]
* backend/s3: Support DynamoDB, IAM, and STS endpoint configurations [GH-19571]
* core: Enhance service discovery error handling and messaging [GH-19589]

BUG FIXES:

* connection/winrm: Set the correct default port when HTTPS is used [GH-19540]
* plugins: GRPC plugins shutdown correctly when Close is called [GH-19629]
* backend/local: Avoid rendering data sources on destroy [GH-19613]
* backend/local: Fix incorrect destroy/update count on apply [GH-19610]
* command/format: Fix rendering of nested blocks during update [GH-19611]
* command/format: Fix rendering of force-new updates [GH-19609]
* helper/schema: Fix setting a set in a list [GH-19552]

## 0.12.0-alpha4 (December 7, 2018)
NOTES:
No changes to terraform; this release is only necessary to fix an incorrect version of the aws provider bundled in alpha3

## 0.12.0-alpha3 (December 6, 2018)

BACKWARDS INCOMPATIBILITIES / NOTES:
* command: Remove `-module-depth` flag from plan, apply, and show. This flag was not widely used and the various updates and improvements to cli output should remove the need for this flag. ([#19267](https://github.com/hashicorp/terraform/issues/19267))
* plugins: The protobuf/grpc package name for the provider protocol was changed from `proto` to `tfplugin5` in preparation for future protocol versioning. This means that plugin binaries built for alpha1 and alpha2 are no longer compatible and will need to be rebuilt. ([#19393](https://github.com/hashicorp/terraform/issues/19393))

IMPROVEMENTS:

* dependencies: upgrading to v21.3.0 of `github.com/Azure/azure-sdk-for-go` ([#19414](https://github.com/hashicorp/terraform/issues/19414))
* dependencies: upgrading to v10.15.4 of `github.com/Azure/go-autorest` ([#19414](https://github.com/hashicorp/terraform/issues/19414))
* backend/azurerm: Fixing a bug where locks couldn't be unlocked ([#19441](https://github.com/hashicorp/terraform/issues/19441))
* backend/azurerm: Support for authenticating via Managed Service Identity ([#19433](https://github.com/hashicorp/terraform/issues/19433))
* backend/azurerm: Support for authenticating using a SAS Token ([#19440](https://github.com/hashicorp/terraform/issues/19440))
* backend/azurerm: Support for custom Resource Manager Endpoints ([#19460](https://github.com/hashicorp/terraform/issues/19460))
* backend/azurerm: Using the proxy from the environment when set ([#19414](https://github.com/hashicorp/terraform/issues/19414))
* backend/azurerm: Deprecating the `arm_` prefix for keys used in the backend configuration ([#19448](https://github.com/hashicorp/terraform/issues/19448))
* command/state: Update and enable the `state show` command ([#19200](https://github.com/hashicorp/terraform/issues/19200))
* command/state: Lock the state when pushing a new state using `state push` ([#19411](https://github.com/hashicorp/terraform/issues/19411))
* backend/remote: Implement the remote enhanced backend ([#19299](https://github.com/hashicorp/terraform/issues/19299))
* backend/remote: Support remote state only usage by dynamically falling back to the local backend ([#19378](https://github.com/hashicorp/terraform/issues/19378))
* backend/remote: Also show Sentinel policy output when there are no changes ([#19403](https://github.com/hashicorp/terraform/issues/19403))
* backend/remote: Add support for the `console`, `graph` and `import` commands ([#19464](https://github.com/hashicorp/terraform/issues/19464))
* backend/remote: Use the new force-unlock API ([#19520](https://github.com/hashicorp/terraform/issues/19520))
* plugin/discovery: Use signing keys from the Terraform Registry when downloading providers. ([#19389](https://github.com/hashicorp/terraform/issues/19389))
* plugin/discovery: Use default `-` namespace alias when fetching available providers from Terraform Registry. ([#19494](https://github.com/hashicorp/terraform/issues/19494))

BUG FIXES:

* command/format: Fix rendering of attribute-agnostic diagnostics ([#19453](https://github.com/hashicorp/terraform/issues/19453))
* core: Fix inconsistent plans when replacing instances. ([#19233](https://github.com/hashicorp/terraform/issues/19233))
* core: Correct handling of unknown values in module outputs during planning and final resolution of them during apply. ([#19237](https://github.com/hashicorp/terraform/issues/19237))
* core: Correct handling of wildcard dependencies when upgrading states ([#19374](https://github.com/hashicorp/terraform/issues/19374))
* core: Fix missing validation of references to non-existing child modules, which was previously resulting in a panic. ([#19487](https://github.com/hashicorp/terraform/issues/19487))
* helper/schema: Don't re-apply schema StateFuncs during apply ([#19536](https://github.com/hashicorp/terraform/issues/19536))
* helper/schema: Allow providers to continue setting and empty string to a default bool value ([#19521](https://github.com/hashicorp/terraform/issues/19521))
* helper/schema: Prevent the insertion of empty diff values when converting legacy diffs ([#19253](https://github.com/hashicorp/terraform/issues/19253))
* helper/schema: Fix timeout parsing during Provider.Diff (plan) ([#19286](https://github.com/hashicorp/terraform/issues/19286))
* helper/schema: Provider arguments set from environment variables now work correctly again, after regressing in the prior 0.12 alphas. ([#19478](https://github.com/hashicorp/terraform/issues/19478))
* helper/schema: For schema attributes that have `Elem` as a nested `schema.Resource`, setting `Optional: true` now forces `MinItems` to be zero, thus mimicking a previously-undocumented behavior that some providers were relying on. ([#19478](https://github.com/hashicorp/terraform/issues/19478))
* helper/schema: Always propagate NewComputed for previously zero value primitive type attributes ([#19548](https://github.com/hashicorp/terraform/issues/19548))
* backend/remote: Fix issues with uploaded configs that contain symlinks ([#19520](https://github.com/hashicorp/terraform/issues/19520))

## 0.12.0-alpha2 (October 30, 2018)

IMPROVEMENTS:

* backend/s3: Support `credential_source` if specified in AWS configuration file ([#19190](https://github.com/hashicorp/terraform/issues/19190))
* command/state: Update and enable the `state mv` command ([#19197](https://github.com/hashicorp/terraform/issues/19197))
* command/state: Update and enable the `state rm` command ([#19178](https://github.com/hashicorp/terraform/issues/19178))

BUG FIXES:

* lang: Fix crash in `lookup` function ([#19161](https://github.com/hashicorp/terraform/issues/19161))
* Hostnames inside module registry source strings may now contain segments that begin with digits, due to an upstream fix in the IDNA parsing library. ([#18039](https://github.com/hashicorp/terraform/issues/18039))
* helper/schema: Fix panic when null values appear for nested blocks ([#19201](https://github.com/hashicorp/terraform/issues/19201))
* helper/schema: Restore handling of the special "timeouts" block in certain resource types. ([#19222](https://github.com/hashicorp/terraform/issues/19222))
* helper/schema: Restore handling of DiffSuppressFunc and StateFunc. ([#19226](https://github.com/hashicorp/terraform/issues/19226))

## 0.12.0-alpha1 (October 19, 2018)

The goal of this release is to give users an early preview of the new language features, and to collect feedback primarily about bugs and usability issues related to the language itself, while the Terraform team addresses the remaining problems. There will be at least one beta and at least one release candidate before final, which should give a more complete impression of how the final v0.12.0 release will behave.

INCOMPATIBILITIES AND NOTES:

The following list contains the most important incompatibillities and notes relative to v0.11.8, but may be incomplete. This alpha release is offered for experimentation purposes only and should not be used to manage real infrastructure. A more complete upgrade guide will be prepared in time for the final v0.12.0 release.

* This release includes a revamped implementation of the configuration language that aims to address a wide array of feedback and known issues with the configuration language handling in prior versions. In order to resolve some ambiguities in the language, the new parser is stricter in some ways about following what was previously just idiomatic usage, and so some unusual constructs will need to be adjusted to be accepted by the new parser.

  The v0.12.0 final release will include a more complete language upgrade guide and a tool that can recognize and automatically upgrade common patterns for the new parser and new idiomatic forms.

* This release introduces new wire protocols for provider and provisioner plugins and a new automatic installation method for provider plugins. At the time of release there are no official plugin releases compatible with these new protocols and so automatic provider installation with `terraform init` is not functional. Instead, the v0.12.0-alpha1 distribution archives contain bundled experimental provider builds for use with the alpha.

* This release introduces new file formats for persisted Terraform state (both on local disk and in remote backends) and for saved plan files. Third-party tools that attempt to parse these files will need to be updated to work with the formats generated by v0.12 releases. Prior to v0.12.0 we will add a new command to obtain a JSON representation of a saved plan intended for outside consumption, but this command is not yet present in v0.12.0-alpha1.

* `terraform validate` now has a smaller scope than before, focusing only on configuration syntax and value type checking. This makes it safe to run e.g. on save in a text editor.

NEW FEATURES:

The overall theme for the v0.12 release is configuration language fixes and improvements. The fixes and improvements are too numerous to list out exhaustively, but the list below covers some highlights:

* **First-class expressions:** Prior to v0.12, expressions could be used only via string interpolation, like `"${var.foo}"`. Expressions are now fully integrated into the language, allowing them to be used directly as argument values, like `ami = var.ami`.

* **`for` expressions:** This new expression construct allows the construction of a list or map by transforming and filtering elements from another list or map. For more information, refer to [the _`for` expressions_ documentation](./website/docs/configuration/expressions.html.md#for-expressions).

* **Dynamic configuration blocks:** For nested configuration blocks accepted as part of a resource configuration, it is now possible to dynamically generate zero or more blocks corresponding to items in a list or map using the special new `dynamic` block construct. This is the official replacement for the common (but buggy) unofficial workaround of treating a block type name as if it were an attribute expecting a list of maps value, which worked occasionally before as a result of some unintended coincidences in the implementation.

* **Generalised "splat" operator:** The `aws_instance.foo.*.id` syntax was previously a special case only for resources with `count` set. It is now an operator within the expression language that can be applied to any list value. There is also an optional new splat variant that allows both index and attribute access operations on each item in the list. For more information, refer to [the _Splat Expressions_ documentation](./website/docs/configuration/expressions.html.md#splat-expressions).

* **Nullable argument values:** It is now possible to use a conditional expression like `var.foo != "" ? var.foo : null` to conditionally leave an argument value unset, whereas before Terraform required the configuration author to provide a specific default value in this case. Assigning `null` to an argument is equivalent to omitting that argument entirely.

* **Rich types in module inputs variables and output values:** Terraform v0.7 added support for returning flat lists and maps of strings, but this is now generalized to allow returning arbitrary nested data structures with mixed types. Module authors can specify a precise expected type for each input variable to allow early validation of caller values.

* **Resource and module object values:** An entire resource or module can now be treated as an object value within expressions, including passing them through input variables and output values to other modules, using an attribute-less reference syntax, like `aws_instance.foo`.

* **Extended template syntax:** The simple interpolation syntax from prior versions is extended to become a simple template language, with support for conditional interpolations and repeated interpolations through iteration. For more information, see [the _String Templates_ documentation](./website/docs/configuration/expressions.html.md#string-templates).

* **`jsondecode` and `csvdecode` interpolation functions:** Due to the richer type system in the new configuration language implementation, we can now offer functions for decoding serialization formats. `jsondecode` is the opposite of `jsonencode`, while `csvdecode` provides a way to load in lists of maps from a compact tabular representation.

* **New Function:** `fileexists` ([#19086](https://github.com/hashicorp/terraform/issues/19086))

IMPROVEMENTS:

* `terraform validate` now accepts an argument `-json` which produces machine-readable output. Please refer to the documentation for this command for details on the format and some caveats that consumers must consider when using this interface. ([#17539](https://github.com/hashicorp/terraform/issues/17539))

* The JSON-based variant of the Terraform language now has a more tightly-specified and reliable mapping to the native syntax variant. In prior versions, certain Terraform configuration features did not function as expected or were not usable via the JSON-based forms. For more information, see [the _JSON Configuration Syntax_ documentation](./website/docs/configuration/syntax-json.html.md).

BUG FIXES:

* The conditional operator `... ? ... : ...` now works with result values of any type and only returns evaluation errors for the chosen result expression, as those familiar with this operator in other languages might expect.

KNOWN ISSUES:

Since v0.12.0-alpha1 is an experimental build, this list is certainly incomplete. Please let us know via GitHub issues if you run into a problem not covered here!

* As noted above, the alpha1 release is bundled with its own special builds of a subset of providers because there are not yet any official upstream releases of providers that are compatible with the new v0.12 provider plugin protocol. Automatic installation of providers with `terraform init` is therefore not functional at the time of release of alpha1.

  Provider developers may wish to try building their plugins against the v0.12-alpha1 tag of Terraform Core to use them with this build. We cannot yet promise that all providers will be buildable in this way and that they will work flawlessly after building. Official releases of all HashiCorp-hosted providers compatible with v0.12 will follow at some point before v0.12.0 final.

* For providers that have required configuration arguments that can be set using environment variables, such as `AWS_REGION` in the `aws` provider, the detection of these environment variables is currently happening too "late" and so Terraform will prompt for these to be entered interactively or generate incorrect error messages saying that they are not set. To work around this, set these arguments inline within the configuration block. In most cases this _does not_ apply to arguments related to API credentials, since most providers declare these ones as optional and then handle the environment variables directly in their own code. The environment variable defaults will be restored before final release.

* There are several error messages in Terraform Core that claim that a problem is caused by a bug in the provider and ask for an issue to be filed against that provider's repository. For this alpha release, we ask that users disregard this advice and report such problems instead within the Terraform Core repository, since they are more likely to be problems with the new protocol version bridge code that is included in the plugin SDK.

* Some secondary Terraform CLI subcommands are not yet updated for this release and will return errors or produce partial results. Please focus most testing and experimentation with this release on the core workflow commands `terraform init`, `terraform validate`, `terraform plan`,  `terraform apply`, and `terraform destroy`.

In addition to the high-level known issues above, please refer also to [the GitHub issues for this alpha release](https://github.com/hashicorp/terraform/issues?utf8=%E2%9C%93&q=is%3Aissue+label%3Av0.12-alpha1+). This list will be updated with new reports throughout the alpha1 period, including workarounds where possible to allow for continued testing. (Issues shown in that list as closed indicate that the problem has been fixed for a future release; it is probably still present in the alpha1 release.)

## 0.11.10 (October 23, 2018)

BUG FIXES:

* backend/local: Do not use backend operation variables ([#19175](https://github.com/hashicorp/terraform/issues/19175))

## 0.11.9 (October 19, 2018)

IMPROVEMENTS:

* backend/remote: Also show policy check output when running a plan ([#19088](https://github.com/hashicorp/terraform/issues/19088))

## 0.11.9-beta1 (October 15, 2018)

IMPROVEMENTS:

* provisioner/chef: Use user:group chown syntax ([#18533](https://github.com/hashicorp/terraform/issues/18533))
* helper/resource: Add `ParallelTest()` to allow opt-in acceptance testing concurrency with `t.Parallel()` ([#18688](https://github.com/hashicorp/terraform/issues/18688))
* backend/manta: Deprecate the `objectName` attribute in favor of the new `object_name` attribute ([#18759](https://github.com/hashicorp/terraform/issues/18759))
* backend/migrations: Migrate existing non-empty default states when the backend only supports named states ([#18760](https://github.com/hashicorp/terraform/issues/18760))
* provider/terraform: `terraform_remote_state` now accepts complex backend configurations ([#18759](https://github.com/hashicorp/terraform/issues/18759))
* backend/remote: Implement the state.Locker interface to support state locking ([#18826](https://github.com/hashicorp/terraform/issues/18826))
* backend/remote: Add initial support for the apply command ([#18950](https://github.com/hashicorp/terraform/issues/18950))
* backend/remote: Ask to cancel pending remote operations when Ctrl-C is pressed ([#18979](https://github.com/hashicorp/terraform/issues/18979))
* backend/remote: Add support for the `-no-color` command line flag ([#19002](https://github.com/hashicorp/terraform/issues/19002))
* backend/remote: Prevent running plan or apply without permissions ([#19012](https://github.com/hashicorp/terraform/issues/19012))
* backend/remote: Add checks for all flags we currently don’t support ([#19013](https://github.com/hashicorp/terraform/issues/19013))
* backend/remote: Allow enhanced backends to pass custom exit codes ([#19014](https://github.com/hashicorp/terraform/issues/19014))
* backend/remote: Properly handle workspaces that auto apply changes ([#19022](https://github.com/hashicorp/terraform/issues/19022))
* backend/remote: Don’t ask questions when `-auto-approve` is set ([#19035](https://github.com/hashicorp/terraform/issues/19035))
* backend/remote: Print status updates while waiting for the run to start ([#19047](https://github.com/hashicorp/terraform/issues/19047))

BUG FIXES:

* backend/azurerm: Update endpoint for Azure Government (SDK Update) ([#18877](https://github.com/hashicorp/terraform/issues/18877))
* backend/migrations: Check all workspaces for existing non-empty states ([#18757](https://github.com/hashicorp/terraform/issues/18757))
* provider/terraform: Always call the backend validation method to prevent a possible panic ([#18759](https://github.com/hashicorp/terraform/issues/18759))
* backend/remote: Take working directories (optional on workspaces) into account ([#18773](https://github.com/hashicorp/terraform/issues/18773))
* backend/remote: Use pagination when retrieving states (workspaces) ([#18817](https://github.com/hashicorp/terraform/issues/18817))
* backend/remote: Add the run ID to associate state when being used in TFE ([#18818](https://github.com/hashicorp/terraform/issues/18818))
* core: Make sure the state is locked before it is used when `(un)tainting` ([#18894](https://github.com/hashicorp/terraform/issues/18894))

## 0.11.8 (August 15, 2018)

NEW FEATURES:

* **New `remote` backend**: Inital release of the `remote` backend for use with Terraform Enterprise and Private Terraform Enterprise ([#18596](https://github.com/hashicorp/terraform/issues/18596))

IMPROVEMENTS:

* cli: display workspace name in apply and destroy commands if not default ([#18253](https://github.com/hashicorp/terraform/issues/18253))
* cli: Remove error on empty outputs when `-json` is set ([#11721](https://github.com/hashicorp/terraform/issues/11721))
* helper/schema: Resources have a new `DeprecationMessage` property that can be set to a string, allowing full resources to be deprecated ([#18286](https://github.com/hashicorp/terraform/issues/18286))
* backend/s3: Allow fallback to session-derived credentials (e.g. session via `AWS_PROFILE` environment variable and shared configuration) ([#17901](https://github.com/hashicorp/terraform/issues/17901))
* backend/s3: Allow usage of `AWS_EC2_METADATA_DISABLED` environment variable ([#17901](https://github.com/hashicorp/terraform/issues/17901))

BUG FIXES:

* config: The `rsadecrypt` interpolation function will no longer include the private key in an error message if it cannot be processed. ([#18333](https://github.com/hashicorp/terraform/issues/18333))
* provisioner/habitat: add missing space for service url ([#18400](https://github.com/hashicorp/terraform/issues/18400))
* backend/s3: Skip extraneous EC2 metadata redirect ([#18570](https://github.com/hashicorp/terraform/issues/18570))

## 0.11.7 (April 10, 2018)

BUG FIXES:

* core: Fix handling of interpolated counts when applying a destroy plan ([#17824](https://github.com/hashicorp/terraform/issues/17824))

PROVIDER SDK CHANGES (not user-facing):

* helper/schema: Invoking `ForceNew` on a key being removed from config during
  diff customization now correctly exposes that key as being removed in the
  updated diff. This prevents diff mismatches under certain circumstances.
  ([#17811](https://github.com/hashicorp/terraform/issues/17811))
* helper/schema: Invoking `ForceNew` during diff customization on its own no
  longer writes any new data to the diff. This prevents writing of new nil to
  zero value diffs for sub-fields of complex lists and sets where a diff did not
  exist before. ([#17811](https://github.com/hashicorp/terraform/issues/17811))

## 0.11.6 (April 5, 2018)

BUG FIXES:

* cli: Don't allow -target without arguments ([#16360](https://github.com/hashicorp/terraform/issues/16360))
* cli: Fix strange formatting of list and map values in the `terraform console` command ([#17714](https://github.com/hashicorp/terraform/issues/17714))
* core: Don't evaluate unused outputs during a full destroy operation ([#17768](https://github.com/hashicorp/terraform/issues/17768))
* core: Fix local and output evaluation when they reference a resource being scaled down to 0 ([#17765](https://github.com/hashicorp/terraform/issues/17765))
* connection/ssh: Retry on authentication failures when the remote service is available before it is completely configured ([#17744](https://github.com/hashicorp/terraform/issues/17744))
* connection/winrm: Get execution errors from winrm commands ([#17788](https://github.com/hashicorp/terraform/issues/17788))
* connection/winrm: Support NTLM authentication ([#17748](https://github.com/hashicorp/terraform/issues/17748))
* provisioner/chef: Fix regression causing connection to be prematurely closed ([#17609](https://github.com/hashicorp/terraform/pull/17609))
* provisioner/habitat: Set channel and builder URL during install, and enable service before start ([#17403](https://github.com/hashicorp/terraform/issues/17403)) ([#17781](https://github.com/hashicorp/terraform/issues/17781))

PROVIDER SDK CHANGES (not user-facing):

* helper/schema: Attribute value is no longer included in error message when `ConflictsWith` keys are used together. ([#17738](https://github.com/hashicorp/terraform/issues/17738))

## 0.11.5 (March 21, 2018)

IMPROVEMENTS:

* provisioner/chef: Allow specifying a channel ([#17355](https://github.com/hashicorp/terraform/issues/17355))

BUG FIXES:

* core: Fix the timeout handling for provisioners ([#17646](https://github.com/hashicorp/terraform/issues/17646))
* core: Ensure that state is unlocked after running console, import, graph or push commands ([#17645](https://github.com/hashicorp/terraform/issues/17645))
* core: Don't open multiple file descriptors for local state files, which would cause reading the state to fail on Windows ([#17636](https://github.com/hashicorp/terraform/issues/17636))

## 0.11.4 (March 15, 2018)

IMPROVEMENTS:

* cli: `terraform state list` now accepts a new argument `-id=...` for filtering resources for display by their remote ids ([#17221](https://github.com/hashicorp/terraform/issues/17221))
* cli: `terraform destroy` now uses the option `-auto-approve` instead of `-force`, for consistency with `terraform apply`. The old flag is preserved for backward-compatibility, but is now deprecated; it will be retained for at least one major release. ([#17218](https://github.com/hashicorp/terraform/issues/17218))
* connection/ssh: Add support for host key verification ([#17354](https://github.com/hashicorp/terraform/issues/17354))
* backend/s3: add support for the cn-northwest-1 region ([#17216](https://github.com/hashicorp/terraform/issues/17216))
* provisioner/local-exec: Allow setting custom environment variables when running commands ([#13880](https://github.com/hashicorp/terraform/issues/13880))
* provisioner/habitat: Detect if hab user exists and only create if necessary ([#17195](https://github.com/hashicorp/terraform/issues/17195))
* provisioner/habitat: Allow custom service name ([#17196](https://github.com/hashicorp/terraform/issues/17196))
* general: https URLs are now supported in the HTTP_PROXY environment variable for URLs interpreted by Terraform Core. (This will not immediately be true for all Terraform provider plugins, since each must upgrade its own HTTP client.) [go1.10:net/http](https://golang.org/doc/go1.10#net/http)

BUG FIXES:

* core: Make sure state is locked during initial refresh ([#17422](https://github.com/hashicorp/terraform/issues/17422))
* core: Halt on fatal provisioner errors, rather than retrying until a timeout ([#17359](https://github.com/hashicorp/terraform/issues/17359))
* core: When handling a forced exit due to multiple interrupts, prevent the process from exiting while the state is being written ([#17323](https://github.com/hashicorp/terraform/issues/17323))
* core: Fix handling of locals and outputs at destroy time ([#17241](https://github.com/hashicorp/terraform/issues/17241))
* core: Fix regression in handling of `count` arguments that refer to `count` attributes from other resources ([#17548](https://github.com/hashicorp/terraform/issues/17548))
* provider/terraform: restore support for the deprecated `environment` argument to the `terraform_remote_state` data source ([#17545](https://github.com/hashicorp/terraform/issues/17545))
* backend/gcs: Report the correct lock ID for GCS state locks ([#17397](https://github.com/hashicorp/terraform/issues/17397))

PROVIDER SDK CHANGES (not user-facing):

* helper/schema: Prevent crash on removal of computed field in CustomizeDiff ([#17261](https://github.com/hashicorp/terraform/issues/17261))
* helper/schema: Allow ResourceDiff.ForceNew on nested fields (avoid crash) ([#17463](https://github.com/hashicorp/terraform/issues/17463))
* helper/schema: Allow `TypeMap` to have a `*schema.Schema` as its `Elem`, for consistency with `TypeSet` and `TypeList` ([#17097](https://github.com/hashicorp/terraform/issues/17097))
* helper/validation: Add ValidateRFC3339TimeString function ([#17484](https://github.com/hashicorp/terraform/issues/17484))

## 0.11.3 (January 31, 2018)

IMPROVEMENTS:

* backend/s3: add support for the eu-west-3 region ([#17193](https://github.com/hashicorp/terraform/issues/17193))


BUG FIXES:

* core: fix crash when an error is encountered during refresh ([#17076](https://github.com/hashicorp/terraform/issues/17076))
* config: fixed crash when module source is invalid ([#17134](https://github.com/hashicorp/terraform/issues/17134))
* config: allow the count pseudo-attribute of a resource to be interpolated into `provisioner` and `connection` blocks without errors ([#17133](https://github.com/hashicorp/terraform/issues/17133))
* backend/s3: allow the workspace name to be a prefix of workspace_key_prefix ([#17086](https://github.com/hashicorp/terraform/issues/17086))
* provisioner/chef: fix crash when validating `use_policyfile` ([#17147](https://github.com/hashicorp/terraform/issues/17147))

## 0.11.2 (January 9, 2018)

BACKWARDS INCOMPATIBILITIES / NOTES:

* backend/gcs: The gcs remote state backend was erroneously creating the state bucket if it didn't exist. This is not the intended behavior of backends, as Terraform cannot track or manage that resource. The target bucket must now be created separately, before using it with Terraform. ([#16865](https://github.com/hashicorp/terraform/issues/16865))

NEW FEATURES:

* **[Habitat](https://www.habitat.sh/) Provisioner** allowing automatic installation of the Habitat agent ([#16280](https://github.com/hashicorp/terraform/issues/16280))

IMPROVEMENTS:

* core: removed duplicate prompts and clarified working when migration backend configurations ([#16939](https://github.com/hashicorp/terraform/issues/16939))
* config: new `rsadecrypt` interpolation function allows decrypting a base64-encoded ciphertext using a given private key. This is particularly useful for decrypting the password for a Windows instance on AWS EC2, but is generic and may find other uses too. ([#16647](https://github.com/hashicorp/terraform/issues/16647))
* config: new `timeadd` interpolation function allows calculating a new timestamp relative to an existing known timestamp. ([#16644](https://github.com/hashicorp/terraform/issues/16644))
* cli: Passing an empty string to `-plugin-dir` during init will remove previously saved paths ([#16969](https://github.com/hashicorp/terraform/issues/16969))
* cli: Module and provider installation (and some other Terraform features) now implement [RFC6555](https://tools.ietf.org/html/rfc6555) when making outgoing HTTP requests, which should improve installation reliability for dual-stack (both IPv4 and IPv6) hosts running on networks that have non-performant or broken IPv6 Internet connectivity by trying both IPv4 and IPv6 connections. ([#16805](https://github.com/hashicorp/terraform/issues/16805))
* backend/s3: it is now possible to disable the region check, for improved compatibility with third-party services that attempt to mimic the S3 API. ([#16757](https://github.com/hashicorp/terraform/issues/16757))
* backend/s3: it is now possible to for the path-based S3 API form, for improved compatibility with third-party services that attempt to mimic the S3 API. ([#17001](https://github.com/hashicorp/terraform/issues/17001))
* backend/s3: it is now possible to use named credentials from the `~/.aws/credentials` file, similarly to the AWS plugin ([#16661](https://github.com/hashicorp/terraform/issues/16661))
* backend/manta: support for Triton RBAC ([#17003](https://github.com/hashicorp/terraform/issues/17003))
* backend/gcs: support for customer-supplied encryption keys for remote state buckets ([#16936](https://github.com/hashicorp/terraform/issues/16936))
* provider/terraform: in `terraform_remote_state`, the argument `environment` is now deprecated in favor of `workspace`. The `environment` argument will be removed in a later Terraform release. ([#16558](https://github.com/hashicorp/terraform/issues/16558))

BUG FIXES:

* config: fixed crash in `substr` interpolation function with invalid offset ([#17043](https://github.com/hashicorp/terraform/issues/17043))
* config: Referencing a count attribute in an output no longer generates a warning ([#16866](https://github.com/hashicorp/terraform/issues/16866))
* cli: Terraform will no longer crash when `terraform plan`, `terraform apply`, and some other commands encounter an invalid provider version constraint in configuration, generating a proper error message instead. ([#16867](https://github.com/hashicorp/terraform/issues/16867))
* backend/gcs: The usage of the GOOGLE_CREDENTIALS environment variable now matches that of the google provider ([#16865](https://github.com/hashicorp/terraform/issues/16865))
* backend/gcs: fixed the locking methodology to avoid "double-locking" issues when used with the `terraform_remote_state` data source ([#16852](https://github.com/hashicorp/terraform/issues/16852))
* backend/s3: the `workspace_key_prefix` can now be an empty string or contain slashes ([#16932](https://github.com/hashicorp/terraform/issues/16932))
* provisioner/salt-masterless: now waits for all of the remote operations to complete before returning ([#16704](https://github.com/hashicorp/terraform/issues/16704))

## 0.11.1 (November 30, 2017)

IMPROVEMENTS:

* modules: Modules can now receive a specific provider configuration in the `providers` map, even if it's only implicitly used ([#16619](https://github.com/hashicorp/terraform/issues/16619))
* config: Terraform will now detect and warn about outputs containing potentially-problematic references to resources with `count` set where the references does not use the "splat" syntax. This identifies situations where an output may [reference a resource with `count = 0`](https://www.terraform.io/upgrade-guides/0-11.html#referencing-attributes-from-resources-with-count-0) even if the `count` expression does not _currently_ evaluate to `0`, allowing the bug to be detected and fixed _before_ the value is later changed to `0` and would thus become an error. **This usage will become a fatal error in Terraform 0.12**. ([#16735](https://github.com/hashicorp/terraform/issues/16735))
* core: A new environment variable `TF_WARN_OUTPUT_ERRORS=1` is supported to opt out of the behavior introduced in 0.11.0 where errors in output expressions halt execution. This restores the previous behavior where such errors are ignored, allowing users to apply problematic configurations without fixing all of the errors. This opt-out will be removed in Terraform 0.12, so it is strongly recommended to use the new warning described in the previous item to detect and fix these problematic expressions. ([#16782](https://github.com/hashicorp/terraform/issues/16782))

BUG FIXES:

* cli: fix crash when subcommands with sub-subcommands are accidentally provided as a single argument, such as `terraform "workspace list"` ([#16789](https://github.com/hashicorp/terraform/issues/16789))

## 0.11.0 (November 16, 2017)

The following list combines the changes from 0.11.0-beta1 and 0.11.0-rc1 to give the full set of changes since 0.10.8. For details on each of the individual pre-releases, please see [the 0.11.0-rc1 CHANGELOG](https://github.com/hashicorp/terraform/blob/v0.11.0-rc1/CHANGELOG.md).

BACKWARDS INCOMPATIBILITIES / NOTES:

The following items give an overview of the incompatibilities and other noteworthy changes in this release. For more details on some of these changes, along with information on how to upgrade existing configurations where needed, see [the v0.11 upgrade guide](https://www.terraform.io/upgrade-guides/0-11.html).

* Output interpolation errors are now fatal. Module configs with unused outputs which contained errors will no longer be valid.
* Module configuration blocks have 2 new reserved attribute names, "providers" and "version". Modules using these as input variables will need to be updated.
* The module provider inheritance rules have changed. Inherited provider configurations will no longer be merged with local configurations, and additional (aliased) provider configurations must be explicitly passed between modules when shared. See [the upgrade guide](https://www.terraform.io/upgrade-guides/0-11.html) for more details.
* The command `terraform apply` with no explicit plan argument is now interactive by default. Specifically, it will show the generated plan and wait for confirmation before applying it, similar to the existing behavior of `terraform destroy`. The behavior is unchanged when a plan file argument is provided, and the previous behavior can be obtained _without_ a plan file by using the `-auto-approve` option.
* The `terraform` provider (that is, the provider that contains the `terraform_remote_state` data source) has been re-incorporated as a built-in provider in the Terraform Core executable. In 0.10 it was split into a separate plugin along with all of the other providers, but this provider uses several internal Terraform Core APIs and so in practice it's been confusing to version that separately from Terraform Core. As a consequence, this provider no longer supports version constraints, and so `version` attributes for this provider in configuration must be removed.
* When remote state is enabled, Terraform will no longer generate a local `terraform.tfstate.backup` file before updating remote state. Previously this file could potentially be used to recover a previous state to help recover after a mistake, but it also caused a potentially-sensitive state file to be generated in an unexpected location that may be inadvertently copied or checked in to version control. With this local backup now removed, we recommend instead relying on versioning or backup mechanisms provided by the backend, such as Amazon S3 versioning or Terraform Enterprise's built-in state history mechanism. (Terraform will still create the local file `errored.tfstate` in the unlikely event that there is an error when writing to the remote backend.)

NEW FEATURES:

* modules: Module configuration blocks now have a "version" attribute, to set a version constraint for modules sourced from a registry. ([#16466](https://github.com/hashicorp/terraform/issues/16466))
* modules: Module configuration blocks now have a "providers" attribute, to map a provider configuration from the current module into a submodule ([#16379](https://github.com/hashicorp/terraform/issues/16379))
* backend/gcs: The gcs remote state backend now supports workspaces and locking.
* backend/manta: The Manta backend now supports workspaces and locking ([#16296](https://github.com/hashicorp/terraform/issues/16296))

IMPROVEMENTS:

* cli: The `terraform apply` command now waits for interactive approval of the generated plan before applying it, unless an explicit plan file is provided. ([#16502](https://github.com/hashicorp/terraform/issues/16502))
* cli: The `terraform version` command now prints out the version numbers of initialized plugins as well as the version of Terraform core, so that they can be more easily shared when opening GitHub Issues, etc. ([#16439](https://github.com/hashicorp/terraform/issues/16439))
* cli: A new `TF_DATA_DIR` environment variable can be used to override the location where Terraform stores the files normally placed in the `.terraform` directory. ([#16207](https://github.com/hashicorp/terraform/issues/16207))
* provider/terraform: now built in to Terraform Core so that it will always have the same backend functionality as the Terraform release it corresponds to. ([#16543](https://github.com/hashicorp/terraform/issues/16543))

BUG FIXES:

* config: Provider config in submodules will no longer be overridden by parent providers with the same name. ([#16379](https://github.com/hashicorp/terraform/issues/16379))
* cli: When remote state is enabled, Terraform will no longer generate a local `terraform.tfstate.backup` file before updating remote state. ([#16464](https://github.com/hashicorp/terraform/issues/16464))
* core: state now includes a reference to the provider configuration most recently used to create or update a resource, so that the same configuration can be used to destroy that resource if its configuration (including the explicit pointer to a provider configuration) is removed ([#16586](https://github.com/hashicorp/terraform/issues/16586))
* core: Module outputs can now produce errors, preventing them from silently propagating through the config. ([#16204](https://github.com/hashicorp/terraform/issues/16204))
* backend/gcs: will now automatically add a slash to the given prefix if not present, since without it the workspace enumeration does not function correctly ([#16585](https://github.com/hashicorp/terraform/issues/16585))

PROVIDER FRAMEWORK CHANGES (not user-facing):

* helper/schema: Loosen validation for 'id' field ([#16456](https://github.com/hashicorp/terraform/issues/16456))

## 0.10.8 (October 25, 2017)

NEW FEATURES:

* **New `etcdv3` backend**, for use with the newer etcd api ([#15680](https://github.com/hashicorp/terraform/issues/15680))
* **New interpolation function `chunklist`**, for spliting a list into a list of lists with specified sublist chunk sizes. ([#15112](https://github.com/hashicorp/terraform/issues/15112))

IMPROVEMENTS:

* backend/s3: Add options to skip AWS validation which allows non-AWS S3 backends ([#15553](https://github.com/hashicorp/terraform/issues/15553))

BUG FIXES:

* command/validate: Respect `-plugin-dir` overridden plugin paths in the `terraform validate` command. ([#15985](https://github.com/hashicorp/terraform/issues/15985))
* provisioner/chef: Clean clients from `chef-vault` when `recreate_client` enabled ([#16357](https://github.com/hashicorp/terraform/issues/16357))
* communicator/winrm: Support the `cacert` option for custom certificate authorities when provisioning over WinRM ([#14783](https://github.com/hashicorp/terraform/issues/14783))

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

----

For earlier versions, see [the changelog as of v0.8.8](https://github.com/hashicorp/terraform/blob/v0.8.8/CHANGELOG.md).
