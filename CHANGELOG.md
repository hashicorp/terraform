## 1.5.6 (Unreleased)

BUG FIXES:

* terraform_remote_state: Fixed a potential unsafe read panic when reading from multiple terraform_remote_state data sources ([#33333](https://github.com/hashicorp/terraform/issues/33333))

## 1.5.5 (August 9, 2023)

* `terraform init`: Fix crash when using invalid configuration in backend blocks. ([#33628](https://github.com/hashicorp/terraform/issues/33628))

## 1.5.4 (July 26, 2023)

BUG FIXES:

* `check` blocks: Fixes crash when nested data sources are within configuration targeted by the terraform import command. ([#33578](https://github.com/hashicorp/terraform/issues/33578))
* `check` blocks: Check blocks now operate in line with other checkable objects by also executing during import operations. ([#33578](https://github.com/hashicorp/terraform/issues/33578))

## 1.5.3 (July 12, 2023)

BUG FIXES:

* core: Terraform could fail to evaluate module outputs when they are used in a provider configuration during a destroy operation ([#33462](https://github.com/hashicorp/terraform/pull/33462))
* backend/consul: When failing to save state, `consul CAS failed with transaction errors` no longer shows an error instance memory address, but an actual error message. ([#33108](https://github.com/hashicorp/terraform/pull/33108))
* plan renderer: Fixes crash when rendering the plan if a relevant attribute contains an integer index specified as a string. ([#33475](https://github.com/hashicorp/terraform/issues/33475))

## 1.5.2 (June 28, 2023)

BUG FIXES:

* configs: Multiple `import` blocks with the same `id` string no longer result in a validation error ([#33434](https://github.com/hashicorp/terraform/issues/33434))

## 1.5.1 (June 21, 2023)

BUG FIXES:

* core: plan validation would fail for providers using nested set attributes with computed object attribute ([#33377](https://github.com/hashicorp/terraform/issues/33377))

## 1.5.0 (June 12, 2023)

NEW FEATURES:

* `check` blocks for validating infrastructure: Module and configuration authors can now write independent check blocks within their configuration to validate assertions about their infrastructure.

    The new independent `check` blocks must specify at least one `assert` block, but possibly many, each one with a `condition` expression and an `error_message` expression matching the existing [Custom Condition Checks](https://developer.hashicorp.com/terraform/language/v1.4.x/expressions/custom-conditions).
    Additionally, check blocks can optionally load a scoped [data source](https://developer.hashicorp.com/terraform/language/v1.4.x/data-sources). Scoped data sources match the existing data sources with the exception that they can only be referenced from within their check block.

    Unlike the existing `precondition` and `postcondition` blocks, Terraform will not halt execution should the scoped data block fail or error or if any of the assertions fail.
    This allows practitioners to continually validate the state of their infrastructure outside the usual lifecycle management cycle.

* `import` blocks for importing infrastructure: Root module authors can now use the `import` block to declare their intent that Terraform adopt an existing resource.

    Import is now a configuration-driven, plannable action, and is processed as part of a normal plan. Running `terraform plan` will show a summary of the resources that Terraform has planned to import, along with any other plan changes.

    The existing `terraform import` CLI command has not been modified.

    This is an early version of the `import` block feature, for which we are actively seeking user feedback to shape future development. The `import` block currently does not support interpolation in the `id` field, which must be a string.

* Generating configuration for imported resources: in conjunction with the `import` block, this feature enables easy templating of configuration when importing existing resources into Terraform. A new flag `-generate-config-out=PATH` is added to `terraform plan`. When this flag is set, Terraform will generate HCL configuration for any resource included in an `import` block that does not already have associated configuration, and write it to a new file at `PATH`. Before applying, review the generated configuration and edit it as necessary.

* Adds a new `plantimestamp` function that returns the timestamp at plan time. This is similar to the `timestamp` function which returns the timestamp at apply time ([#32980](https://github.com/hashicorp/terraform/pull/32980)).
* Adds a new `strcontains` function that checks whether a given string contains a given substring. ([#33069](https://github.com/hashicorp/terraform/issues/33069))


UPGRADE NOTES:

* This is the last version of Terraform for which macOS 10.13 High Sierra or 10.14 Mojave are officially supported. Future Terraform versions may not function correctly on these older versions of macOS.
* This is the last version of Terraform for which Windows 7, 8, Server 2008, and Server 2012 are supported by Terraform's main implementation language, Go. We already ended explicit support for versions earlier than Windows 10 in Terraform v0.15.0, but future Terraform versions may malfunction in more significant ways on these older Windows versions.
* On Linux (and some other non-macOS Unix platforms we don't officially support), Terraform will now notice the `trust-ad` option in `/etc/resolv.conf` and, if set, will set the "authentic data" option in outgoing DNS requests in order to better match the behavior of the GNU libc resolver.

    Terraform does not pay any attention to the corresponding option in responses, but some DNSSEC-aware recursive resolvers return different responses when the request option isn't set. This should therefore avoid some potential situations where a DNS request from Terraform might get a different response than a similar request from other software on your system.

ENHANCEMENTS:

* Terraform CLI's local operations mode will now attempt to persist state snapshots to the state storage backend periodically during the apply step, thereby reducing the window for lost data if the Terraform process is aborted unexpectedly. ([#32680](https://github.com/hashicorp/terraform/issues/32680))
* If Terraform CLI receives SIGINT (or its equivalent on non-Unix platforms) during the apply step then it will immediately try to persist the latest state snapshot to the state storage backend, with the assumption that a graceful shutdown request often typically followed by a hard abort some time later if the graceful shutdown doesn't complete fast enough. ([#32680](https://github.com/hashicorp/terraform/issues/32680))
* `pg` backend: Now supports the `PG_CONN_STR`, `PG_SCHEMA_NAME`, `PG_SKIP_SCHEMA_CREATION`, `PG_SKIP_TABLE_CREATION` and `PG_SKIP_INDEX_CREATION` environment variables. ([#33045](https://github.com/hashicorp/terraform/issues/33045))

BUG FIXES:

* `terraform init`: Fixed crash with invalid blank module name. ([#32781](https://github.com/hashicorp/terraform/issues/32781))
* `moved` blocks: Fixed a typo in the error message that Terraform raises when you use `-target` to exclude an object that has been moved. ([#33149](https://github.com/hashicorp/terraform/issues/33149))

## Previous Releases

For information on prior major and minor releases, see their changelogs:

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
