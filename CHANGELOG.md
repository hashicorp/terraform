## 1.8.0-rc1 (March 20 2024)

If you are upgrading from Terraform v1.7 or earlier, please refer to
[the Terraform v1.8 Upgrade Guide](https://developer.hashicorp.com/terraform/language/v1.8.x/upgrade-guides).

NEW FEATURES:

* Providers can now offer functions which can be used from within the Terraform configuration language.

    The syntax for calling a provider-contributed function is `provider::provider_name::function_name()`. ([#34394](https://github.com/hashicorp/terraform/issues/34394))
* Providers can now transfer the ownership of a remote object between resources of different types, for situations where there are two different resource types that represent the same remote object type.

    This extends the `moved` block behavior to support moving between two resources of different types only if the provider for the target resource type declares that it can convert from the source resource type. Refer to provider documentation for details on which pairs of resource types are supported.
* New `issensitive` function returns true if the given value is marked as sensitive.

ENHANCEMENTS:

* `terraform test`: File-level variables can now refer to global variables. ([#34699](https://github.com/hashicorp/terraform/issues/34699))
* When generating configuration based on `import` blocks, Terraform will detect strings that contain valid JSON syntax and generate them as calls to the `jsonencode` function, rather than generating a single string. This is primarily motivated by readability, but might also be useful if you need to replace part of the literal value with an expression as you generalize your module beyond the one example used for importing.
* `terraform plan` now uses a different presentation for describing changes to lists where the old and new lists have the same length. It now compares the elements with correlated indices and shows a separate diff for each one, rather than trying to show a diff for the list as a whole. The behavior is unchanged for lists of different lengths.
* `terraform providers lock` accepts a new boolean option `-enable-plugin-cache`. If specified, and if a [global plugin cache](https://developer.hashicorp.com/terraform/cli/config/config-file#provider-plugin-cache) is configured, Terraform will use the cache in the provider lock process. ([#34632](https://github.com/hashicorp/terraform/issues/34632))
* built-in "terraform" provider: new `decode_tfvars`, `encode_tfvars`, and `encode_expr` functions, for unusual situations where it's helpful to manually generate or read from Terraform's "tfvars" format. ([#34718](https://github.com/hashicorp/terraform/issues/34718))
* `terraform show`'s JSON rendering of a plan now includes two explicit flags `"applyable"` and `"complete"`, which both summarize characteristics of a plan that were previously only inferrable by consumers replicating some of Terraform Core's own logic. ([#34642](https://github.com/hashicorp/terraform/issues/34642))

    `"applyable"` means that it makes sense for a wrapping automation to offer to apply this plan.

    `"complete"` means that applying this plan is expected to achieve convergence between desired and actual state. If this flag is present and set to `false` then wrapping automations should ideally encourage an operator to run another plan/apply round to continue making progress toward convergence.

BUG FIXES:

* core: Sensitive values will now be tracked more accurately in state and plans, preventing unexpected updates with no apparent changes. ([#34567](https://github.com/hashicorp/terraform/issues/34567))
* core: Fix incorrect error message when using in invalid `iterator` argument within a dynamic block. ([#34751](https://github.com/hashicorp/terraform/issues/34751))
* core: Fixed edge-case bug that could cause loss of floating point precision when round-tripping due to incorrectly using a MessagePack integer to represent a large non-integral number. ([#24576](https://github.com/hashicorp/terraform/issues/24576))
* config: Converting from an unknown map value to an object type now correctly handles the situation where the map element type disagrees with an optional attribute of the target type, since when a map value is unknown we don't yet know which keys it has and thus cannot predict what subset of the elements will get converted as attributes in the resulting object. ([#34756](https://github.com/hashicorp/terraform/issues/34756))
* cloud: Fixed unparsed color codes in policy failure error messages. ([#34473](https://github.com/hashicorp/terraform/issues/34473))

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
