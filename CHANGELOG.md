## 1.8.0 (Unreleased)

UPGRADE NOTES:

* The first plan after upgrading may show resource updates with no apparent changes if `-refresh-only` or `-refresh=false` is used. The fix introduced for #34567 may require rewriting the state for some resources, which will be done automatically during the first normal plan and apply operation.

NEW FEATURES:

* Providers can now implement functions which can be used from within the Terraform configuration language. The syntax for calling a provider supplied function is `provider::provider_name::function_name()`. [GH-34394]
* Providers can now implement move operations between resource types, both from resource types defined by the provider and defined by other providers. Check provider documentation for supported cross-resource-type moves.
* `issensitive` function added to detect if a value is marked as sensitive

BUG FIXES:

* core: Sensitive values will now be tracked more accurately in state and plans, preventing unexpected updates with no apparent changes [GH-34567]

EXPERIMENTS:

Experiments are only enabled in alpha releases of Terraform CLI. The following features are not yet available in stable releases.

* `terraform test` accepts a new option `-junit-xml=FILENAME`. If specified, and if the test configuration is valid enough to begin executing, then Terraform writes a JUnit XML test result report to the given filename, describing similar information as included in the normal test output. (#34291)
* The new command `terraform rpcapi` exposes some Terraform Core functionality through an RPC interface compatible with [`go-plugin`](https://github.com/hashicorp/go-plugin). The exact RPC API exposed here is currently subject to change at any time, because it's here primarily as a vehicle to support the [Terraform Stacks](https://www.hashicorp.com/blog/terraform-stacks-explained) private preview and so will be broken if necessary to respond to feedback from private preview participants, or possibly for other reasons. Do not use this mechanism yet outside of Terraform Stacks private preview.
* The [language-level experiment](https://developer.hashicorp.com/terraform/language/settings#experimental-language-features) `unknown_instances` permits `count` and `for_each` arguments in `module`, `resource`, and `data` blocks to have unknown values.

    This is at an early stage and so currently setting these arguments to unknown values will only yield broken behavior, and so it's not yet useful to participate in this experiment. Future work will improve support for this new possibility, gradually making this experiment viable.

## Previous Releases

For information on prior major and minor releases, see their changelogs:

* [v1.7](https://github.com/hashicorp/terraform/blob/v1.7/CHANGELOG.md)
* [v1.6](https://github.com/hashicorp/terraform/blob/v1.6/CHANGELOG.md)
* [v1.5](https://github.com/hashicorp/terraform/blob/v1.5/CHANGELOG.md)
* [v1.4](https://github.com/hashicorp/terraform/blob/v1.4/CHANGELOG.md)
* [v1.3](https://github.com/hashicorp/terraform/blob/v1.3/CHANGELOG.md)
* [v1.2](https://github.com/hashicorp/terraform/blob/v1.2/CHANGELOG.md)
* [v1.1](https://github.com/hashicorp/terraform/blob/v1.1/CHANGELOG.md)
* [v1.0](https://github.com/hashicorp/terraform/blob/v1.0/CHANGELOG.md)
* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
