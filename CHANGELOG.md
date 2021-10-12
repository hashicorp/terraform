## 1.0.9 (Unreleased)

BUG FIXES:

core: Fix panic when planning new resources with nested object attributes [GH-29701]
core: Do not refresh deposed instances when the provider is not configured during destroy [GH-29720]

## 1.0.8 (September 29, 2021)

BUG FIXES:

* cli: Check `required_version` as early as possibly during `init` so that version incompatibility can be reported before errors about new syntax ([#29665](https://github.com/hashicorp/terraform/issues/29665))
* core: Don't plan to remove orphaned resource instances in refresh-only plans ([#29640](https://github.com/hashicorp/terraform/issues/29640))

## 1.0.7 (September 15, 2021)

BUG FIXES:

* core: Remove check for computed attributes which is no longer valid with optional structural attributes ([#29563](https://github.com/hashicorp/terraform/issues/29563))
* core: Prevent object types with optional attributes from being instantiated as concrete values, which can lead to failures in type comparison ([#29559](https://github.com/hashicorp/terraform/issues/29559))
* core: Empty containers in the configuration were not planned correctly when used with optional structural attributes ([#29580](https://github.com/hashicorp/terraform/issues/29580))

## 1.0.6 (September 03, 2021)

ENHANCEMENTS:

* backend/s3: Improve SSO handling and add new endpoints in the AWS SDK ([#29017](https://github.com/hashicorp/terraform/issues/29017))

BUG FIXES:

* cli: Suppress confirmation prompt when initializing with the `-force-copy` flag and migrating state between multiple workspaces. ([#29438](https://github.com/hashicorp/terraform/issues/29438))
* cli: Update tencentcount dependency versions to fix errors when building from source ([#29445](https://github.com/hashicorp/terraform/issues/29445))
* core: Fix panic while handling computed attributes within nested objects, and improve plan validation for unknown values ([#29482](https://github.com/hashicorp/terraform/issues/29482))

## 1.0.5 (August 18, 2021)

BUG FIXES:

* json-output: Add an output change summary message as part of the `terraform plan -json` structured logs, bringing this format into parity with the human-readable UI. ([#29312](https://github.com/hashicorp/terraform/issues/29312))
* core: Handle null nested single attribute values ([#29411](https://github.com/hashicorp/terraform/issues/29411))
* cli: Fix crash when planning a diff between null and empty sets in nested attributes ([#29398](https://github.com/hashicorp/terraform/issues/29398))
* cli: Fix crash when planning a new resource containing a set of nested object attributes ([#29398](https://github.com/hashicorp/terraform/issues/29398))
* cli: Fix crash when displaying a resource diff where a possibly identifying attribute is sensitive ([#29397](https://github.com/hashicorp/terraform/issues/29397))
* cli: Fix crash when a diff with unknown nested map attributes ([#29410](https://github.com/hashicorp/terraform/issues/29410))
* config: Fix handling of dynamically types arguments in `formatlist`, ensuring the correct resulting type. ([#29408](https://github.com/hashicorp/terraform/issues/29408))
* config: Floating point operations like `floor` and `ceil` can no longer mutate their arguments. ([#29408](https://github.com/hashicorp/terraform/issues/29408))

## 1.0.4 (August 04, 2021)


BUG FIXES:

* backend/consul: Fix a bug where the state value may be too large for consul to accept ([#28838](https://github.com/hashicorp/terraform/issues/28838))
* cli: Fixed a crashing bug with some edge-cases when reporting syntax errors that happen to be reported at the position of a newline. ([#29048](https://github.com/hashicorp/terraform/issues/29048))

## 1.0.3 (July 21, 2021)

ENHANCEMENTS

* `terraform plan`: The JSON logs (`-json` option) will now include `resource_drift`, showing changes detected outside of Terraform during the refresh step. ([#29072](https://github.com/hashicorp/terraform/issues/29072))
* core: The automatic provider installer will now accept providers that are recorded in their registry as using provider protocol version 6. ([#29153](https://github.com/hashicorp/terraform/issues/29153))
* backend/etcdv3: New argument `max_request_bytes` allows larger requests and for the client, to match the server request limit. ([#28078](https://github.com/hashicorp/terraform/issues/28078))

BUG FIXES:

* `terraform plan`: Will no longer panic when trying to render null maps. ([#29207](https://github.com/hashicorp/terraform/issues/29207))
* backend/pg: Prevent the creation of multiple workspaces with the same name. ([#29157](https://github.com/hashicorp/terraform/issues/29157))
* backend/oss: STS auth is now supported. ([#29167](https://github.com/hashicorp/terraform/issues/29167))
* config: Dynamic blocks with unknown for_each values were not being validated. Ensure block attributes are valid even when the block is unknown ([#29208](https://github.com/hashicorp/terraform/issues/29208))
* config: Unknown values in string templates could lose sensitivity, causing the planned change to be inaccurate ([#29208](https://github.com/hashicorp/terraform/issues/29208))

## 1.0.2 (July 07, 2021)

BUG FIXES:

* `terraform show`: Fix crash when rendering JSON plan with sensitive values in state ([#29049](https://github.com/hashicorp/terraform/issues/29049))
* config: The `floor` and `ceil` functions no longer lower the precision of arguments to what would fit inside a 64-bit float, instead preserving precision in a similar way as most other arithmetic functions. ([#29110](https://github.com/hashicorp/terraform/issues/29110))
* config: The `flatten` function was incorrectly treating null values of an unknown type as if they were unknown values. Now it will treat them the same as any other non-list/non-tuple value, flattening them down into the result as-is. ([#29110](https://github.com/hashicorp/terraform/issues/29110))

## 1.0.1 (June 24, 2021)

ENHANCEMENTS:

* `terraform show`: The JSON plan output now indicates which state values are sensitive. ([#28889](https://github.com/hashicorp/terraform/issues/28889))
* cli: The macOS builds will now resolve hostnames using the system's DNS resolver, rather than the Go library's (incomplete) emulation of it. In particular, this will allow for the more complex resolver configurations often created by VPN clients on macOS, such as when a particular domain must be resolved using different nameservers while VPN connection is active.

BUG FIXES:

* `terraform show`: Fix crash with deposed instances in json plan output. ([#28922](https://github.com/hashicorp/terraform/issues/28922))
* `terraform show`: Fix an issue where the JSON configuration representation was missing fully-unwrapped references. ([#28884](https://github.com/hashicorp/terraform/issues/28884))
* `terraform show`: Fix JSON plan resource drift to remove unchanged resources. ([#28975](https://github.com/hashicorp/terraform/issues/28975))
* core: Fix crash when provider modifies and unknown block during plan. ([#28941](https://github.com/hashicorp/terraform/issues/28941))
* core: Diagnostic context was missing for some errors when validating blocks. ([#28979](https://github.com/hashicorp/terraform/issues/28979))
* core: Fix crash when calling `setproduct` with unknown values. ([#28984](https://github.com/hashicorp/terraform/issues/28984))
* backend/remote: Fix faulty Terraform Cloud version check when migrating state to the remote backend with multiple local workspaces. ([#28864](https://github.com/hashicorp/terraform/issues/28864))

## 1.0.0 (June 08, 2021)

Terraform v1.0 is an unusual release in that its primary focus is on stability, and it represents the culmination of several years of work in previous major releases to make sure that the Terraform language and internal architecture will be a suitable foundation for forthcoming additions that will remain backward compatible.

Terraform v1.0.0 intentionally has no significant changes compared to Terraform v0.15.5. You can consider the v1.0 series as a direct continuation of the v0.15 series; we do not intend to issue any further releases in the v0.15 series, because all of the v1.0 releases will be only minor updates to address bugs.

For all future minor releases with major version 1, we intend to preserve backward compatibility as described in detail in [the Terraform v1.0 Compatibility Promises](https://www.terraform.io/docs/language/v1-compatibility-promises.html). The later Terraform v1.1.0 will, therefore, be the first minor release with new features that we will implement with consideration of those promises.

## Previous Releases

For information on prior major releases, see their changelogs:

* [v0.15](https://github.com/hashicorp/terraform/blob/v0.15/CHANGELOG.md)
* [v0.14](https://github.com/hashicorp/terraform/blob/v0.14/CHANGELOG.md)
* [v0.13](https://github.com/hashicorp/terraform/blob/v0.13/CHANGELOG.md)
* [v0.12](https://github.com/hashicorp/terraform/blob/v0.12/CHANGELOG.md)
* [v0.11 and earlier](https://github.com/hashicorp/terraform/blob/v0.11/CHANGELOG.md)
